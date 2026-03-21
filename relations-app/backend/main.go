// main.go - обновленный с nested regionsByCountry
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

var db *sql.DB

func main() {
	// Подключение к БД
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = fmt.Sprintf(
			"user=%s dbname=%s password=%s host=%s port=%s sslmode=disable",
			os.Getenv("DB_USER"),
			os.Getenv("DB_NAME"),
			os.Getenv("DB_PASSWORD"),
			os.Getenv("DB_HOST"),
			os.Getenv("DB_PORT"),
		)
	}

	var err error
	db, err = sql.Open("postgres", connStr)
	if err := createDb(db); err != nil {
		log.Fatal("Не удалось инициализировать базу данных:", err)
	}
	if err != nil {
		log.Fatal("Не удалось подключиться к базе данных:", err)
	}

	router := mux.NewRouter()
	router.HandleFunc("/api/start", startHandler).Methods("GET")
	router.HandleFunc("/api/submit-form", submitHandler).Methods("POST")
	router.HandleFunc("/api/statistics", statsHandler).Methods("GET")
	router.HandleFunc("/api/regions", regionsHandler).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Println("Server starting on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, corsMiddleware(router)))
}

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func startHandler(w http.ResponseWriter, r *http.Request) {
	// Загружаем страны из БД
	rows, err := db.Query("SELECT name FROM countries ORDER BY name")
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var countries []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		countries = append(countries, name)
	}

	data := map[string][]string{
		"countries":  countries,
		"educations": {"Среднее оконченное", "Среднее профессиональное", "Высшее неоконченное", "Высшее оконченное", "Учёная степень"},
		"statuses":   {"Женат/Замужем", "Состою в романтических отношениях", "Одинок(а)"},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

type SubmitData struct {
	Gender              string `json:"gender"`
	Age                 int    `json:"age"`
	Education           string `json:"education"`
	Country             string `json:"country"`
	Region              string `json:"region"`
	RelationshipStatus  string `json:"relationship_status"`
	Consent             bool   `json:"consent"`
	PositiveTraits      string `json:"positive_traits"`
	NegativeTraits      string `json:"negative_traits"`
	OverallSatisfaction int    `json:"overall_satisfaction"`
}

func submitHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	var data SubmitData
	if err := json.Unmarshal(body, &data); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Валидация
	if data.Age < 18 || data.Age > 100 || !data.Consent {
		http.Error(w, "Invalid data", http.StatusBadRequest)
		return
	}

	// Находим country_id по name
	var countryID int
	err = db.QueryRow("SELECT id FROM countries WHERE name = $1", data.Country).Scan(&countryID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Invalid country", http.StatusBadRequest)
		} else {
			http.Error(w, "DB error: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Находим region_id по name и country_id
	var regionID int
	err = db.QueryRow(
		"SELECT id FROM regions WHERE name = $1 AND country_id = $2",
		data.Region, countryID,
	).Scan(&regionID)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Указан неверный регион для выбранной страны", http.StatusBadRequest)
		} else {
			http.Error(w, "Ошибка базы данных (регион): "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Рассчёт score
	score, err := calculateScore([]byte(data.PositiveTraits), []byte(data.NegativeTraits), data.OverallSatisfaction)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Сохранение в user_profiles
	var profileID int
	err = db.QueryRow(`
		INSERT INTO user_profiles (gender, age, education, country_id, region_id, relationship_status, consent)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id
	`, data.Gender, data.Age, data.Education, countryID, regionID, data.RelationshipStatus, data.Consent).Scan(&profileID)
	if err != nil {
		http.Error(w, "DB error in profiles: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Сохранение в test_results
	_, err = db.Exec(`
		INSERT INTO test_results (profile_id, positive_traits, negative_traits, overall_satisfaction, calculated_score)
		VALUES ($1, $2, $3, $4, $5)
	`, profileID, data.PositiveTraits, data.NegativeTraits, data.OverallSatisfaction, score)
	if err != nil {
		http.Error(w, "DB error in results: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Возвращает результат
	result := map[string]interface{}{
		"score": score,
		"text":  getScoreText(score),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Вспомогательная функция с текстом результата
func getScoreText(score float64) string {
	if score > 80 {
		return "Ваши отношения характеризуются высокой структурной целостностью, сбалансированными положительными чертами и минимальным негативным влиянием. Между вами царит глубокая гармония, взаимопонимание и ощущение надёжного «мы». Это тот редкий тип связи, который не только радует, но и даёт силы обоим партнёрам."
	} else if score > 60 {
		return "Ваши отношения обладают крепкой структурной целостностью и заметным преобладанием положительных черт. Есть отдельные зоны, где возникают напряжение или недопонимание, но они не разрушают общую картину. С небольшими усилиями и вниманием связь может стать ещё более устойчивой и счастливой."
	} else if score > 40 {
		return "Ваши отношения имеют умеренную структурную целостность. Положительные и негативные черты находятся примерно в равновесии. Связь держится, но часто требует энергии и компромиссов. Сейчас хороший момент, чтобы целенаправленно поработать над слабыми местами — это даст заметный рост качества отношений."
	} else if score > 20 {
		return "Ваши отношения показывают сниженную структурную целостность. Негативные факторы заметно перевешивают положительные. Многие важные потребности остаются неудовлетворёнными, появляется усталость и ощущение дистанции. Без осознанных изменений ситуация может ухудшиться."
	} else {
		return "Ваши отношения находятся в состоянии низкой структурной целостности с преобладанием разрушительных факторов. Положительные черты почти полностью подавлены негативом, что создаёт серьёзный эмоциональный дискомфорт для обоих."
	}
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	stats := make(map[string]interface{})

	// Статистика по gender
	genderRows, err := db.Query(`
    SELECT up.gender, AVG(tr.calculated_score) as avg_score
    FROM user_profiles up
    JOIN test_results tr ON up.id = tr.profile_id
    GROUP BY up.gender
  `)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer genderRows.Close()

	genderStats := make(map[string]float64)
	for genderRows.Next() {
		var gender string
		var avg float64
		genderRows.Scan(&gender, &avg)
		genderStats[gender] = avg
	}
	stats["gender"] = genderStats

	// Статистика по country
	countryRows, err := db.Query(`
    SELECT c.name, AVG(tr.calculated_score) as avg_score
    FROM user_profiles up
    JOIN countries c ON up.country_id = c.id
    JOIN test_results tr ON up.id = tr.profile_id
    GROUP BY c.name
  `)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer countryRows.Close()

	countryStats := make(map[string]float64)
	for countryRows.Next() {
		var name string
		var avg float64
		countryRows.Scan(&name, &avg)
		countryStats[name] = avg
	}
	stats["country"] = countryStats

	// Статистика по education
	educationRows, err := db.Query(`
    SELECT up.education, AVG(tr.calculated_score) as avg_score
    FROM user_profiles up
    JOIN test_results tr ON up.id = tr.profile_id
    GROUP BY up.education
  `)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer educationRows.Close()

	educationStats := make(map[string]float64)
	for educationRows.Next() {
		var education string
		var avg float64
		educationRows.Scan(&education, &avg)
		educationStats[education] = avg
	}
	stats["education"] = educationStats

	// Статистика по регионам, сгруппированная по стране
	regionRows, err := db.Query(`
    SELECT c.name AS country, r.name AS region, AVG(tr.calculated_score) as avg_score
    FROM user_profiles up
    JOIN countries c ON up.country_id = c.id
    JOIN regions r ON up.region_id = r.id
    JOIN test_results tr ON up.id = tr.profile_id
    GROUP BY c.name, r.name
  `)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer regionRows.Close()

	regionsByCountry := make(map[string]map[string]float64)
	for regionRows.Next() {
		var country, region string
		var avg float64
		regionRows.Scan(&country, &region, &avg)
		if _, ok := regionsByCountry[country]; !ok {
			regionsByCountry[country] = make(map[string]float64)
		}
		regionsByCountry[country][region] = avg
	}
	stats["regionsByCountry"] = regionsByCountry

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// Новый обработчик — возвращает регионы для выбранной страны
func regionsHandler(w http.ResponseWriter, r *http.Request) {
	countryName := r.URL.Query().Get("country")
	if countryName == "" {
		http.Error(w, "country parameter is required", http.StatusBadRequest)
		return
	}

	var countryID int
	err := db.QueryRow("SELECT id FROM countries WHERE name = $1", countryName).Scan(&countryID)
	if err != nil {
		http.Error(w, "Invalid country", http.StatusBadRequest)
		return
	}

	rows, err := db.Query("SELECT name FROM regions WHERE country_id = $1 ORDER BY name", countryID)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var regions []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		regions = append(regions, name)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]string{"regions": regions})
}
