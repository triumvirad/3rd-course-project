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
	"time"

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
	router.HandleFunc("/api/seed", seedHandler).Methods("GET")

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
	if score > 100 {
		score = 100
	}
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

func seedHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Очищаем таблицы перед вставкой новых данных
	_, err := db.Exec("DELETE FROM test_results")
	if err != nil {
		http.Error(w, "Ошибка очистки test_results: "+err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = db.Exec("DELETE FROM user_profiles")
	if err != nil {
		http.Error(w, "Ошибка очистки user_profiles: "+err.Error(), http.StatusInternalServerError)
		return
	}

	type FakeEntry struct {
		Gender             string
		Age                int
		Education          string
		Country            string
		Region             string
		RelationshipStatus string
		PositiveTraits     string // строка JSON
		NegativeTraits     string // строка JSON
		Overall            int
	}

	entries := []FakeEntry{
		{"male", 28, "Высшее оконченное", "Россия", "Москва", "Женат/Замужем",
			`[{"trait":"Надёжность","rank":1,"score":95},{"trait":"Юмор","rank":2,"score":82},{"trait":"Поддержка","rank":3,"score":90},{"trait":"Интеллект","rank":4,"score":88},{"trait":"Забота","rank":5,"score":85}]`,
			`[{"trait":"Ревность","rank":1,"score":35},{"trait":"Лень","rank":2,"score":40},{"trait":"Упрямство","rank":3,"score":28},{"trait":"Забывчивость","rank":4,"score":32},{"trait":"Критика","rank":5,"score":25}]`, 84},

		{"female", 25, "Высшее неоконченное", "Россия", "Санкт-Петербург", "Состою в романтических отношениях",
			`[{"trait":"Эмпатия","rank":1,"score":93},{"trait":"Красота","rank":2,"score":90},{"trait":"Страсть","rank":3,"score":87},{"trait":"Честность","rank":4,"score":92},{"trait":"Оптимизм","rank":5,"score":85}]`,
			`[{"trait":"Ревность","rank":1,"score":45},{"trait":"Опоздания","rank":2,"score":38},{"trait":"Эгоизм","rank":3,"score":32},{"trait":"Критика","rank":4,"score":30},{"trait":"Молчаливость","rank":5,"score":25}]`, 76},

		{"male", 34, "Среднее профессиональное", "Россия", "Новосибирская область", "Женат/Замужем",
			`[{"trait":"Стабильность","rank":1,"score":91},{"trait":"Надёжность","rank":2,"score":89},{"trait":"Юмор","rank":3,"score":84},{"trait":"Поддержка","rank":4,"score":88},{"trait":"Забота","rank":5,"score":82}]`,
			`[{"trait":"Лень","rank":1,"score":48},{"trait":"Ревность","rank":2,"score":42},{"trait":"Забывчивость","rank":3,"score":35},{"trait":"Упрямство","rank":4,"score":30},{"trait":"Критика","rank":5,"score":28}]`, 68},

		{"female", 31, "Учёная степень", "Россия", "Екатеринбург", "Женат/Замужем",
			`[{"trait":"Мудрость","rank":1,"score":96},{"trait":"Эмпатия","rank":2,"score":92},{"trait":"Интеллект","rank":3,"score":94},{"trait":"Честность","rank":4,"score":90},{"trait":"Поддержка","rank":5,"score":88}]`,
			`[{"trait":"Перфекционизм","rank":1,"score":25},{"trait":"Ревность","rank":2,"score":20},{"trait":"Критика","rank":3,"score":30},{"trait":"Опоздания","rank":4,"score":22},{"trait":"Эмоциональность","rank":5,"score":18}]`, 91},

		{"male", 29, "Высшее оконченное", "Россия", "Краснодарский край", "Состою в романтических отношениях",
			`[{"trait":"Страсть","rank":1,"score":90},{"trait":"Юмор","rank":2,"score":88},{"trait":"Надёжность","rank":3,"score":85},{"trait":"Забота","rank":4,"score":87},{"trait":"Оптимизм","rank":5,"score":82}]`,
			`[{"trait":"Ревность","rank":1,"score":40},{"trait":"Лень","rank":2,"score":45},{"trait":"Забывчивость","rank":3,"score":38},{"trait":"Упрямство","rank":4,"score":32},{"trait":"Критика","rank":5,"score":28}]`, 72},

		{"female", 26, "Высшее неоконченное", "Россия", "Ростовская область", "Одинок(а)",
			`[{"trait":"Красота","rank":1,"score":92},{"trait":"Эмпатия","rank":2,"score":89},{"trait":"Страсть","rank":3,"score":86},{"trait":"Честность","rank":4,"score":90},{"trait":"Юмор","rank":5,"score":84}]`,
			`[{"trait":"Эгоизм","rank":1,"score":35},{"trait":"Ревность","rank":2,"score":42},{"trait":"Опоздания","rank":3,"score":30},{"trait":"Критика","rank":4,"score":28},{"trait":"Молчаливость","rank":5,"score":25}]`, 74},

		{"male", 42, "Высшее оконченное", "Россия", "Татарстан", "Женат/Замужем",
			`[{"trait":"Стабильность","rank":1,"score":94},{"trait":"Надёжность","rank":2,"score":92},{"trait":"Поддержка","rank":3,"score":90},{"trait":"Интеллект","rank":4,"score":88},{"trait":"Забота","rank":5,"score":85}]`,
			`[{"trait":"Упрямство","rank":1,"score":38},{"trait":"Лень","rank":2,"score":45},{"trait":"Ревность","rank":3,"score":30},{"trait":"Забывчивость","rank":4,"score":35},{"trait":"Критика","rank":5,"score":25}]`, 80},

		{"female", 33, "Высшее оконченное", "Россия", "Башкортостан", "Женат/Замужем",
			`[{"trait":"Эмпатия","rank":1,"score":91},{"trait":"Честность","rank":2,"score":89},{"trait":"Поддержка","rank":3,"score":87},{"trait":"Оптимизм","rank":4,"score":85},{"trait":"Красота","rank":5,"score":82}]`,
			`[{"trait":"Ревность","rank":1,"score":32},{"trait":"Перфекционизм","rank":2,"score":28},{"trait":"Опоздания","rank":3,"score":35},{"trait":"Критика","rank":4,"score":25},{"trait":"Эгоизм","rank":5,"score":20}]`, 82},

		{"male", 30, "Среднее профессиональное", "Россия", "Московская область", "Состою в романтических отношениях",
			`[{"trait":"Надёжность","rank":1,"score":88},{"trait":"Юмор","rank":2,"score":85},{"trait":"Забота","rank":3,"score":90},{"trait":"Стабильность","rank":4,"score":84},{"trait":"Интеллект","rank":5,"score":80}]`,
			`[{"trait":"Лень","rank":1,"score":50},{"trait":"Ревность","rank":2,"score":40},{"trait":"Забывчивость","rank":3,"score":38},{"trait":"Упрямство","rank":4,"score":35},{"trait":"Критика","rank":5,"score":30}]`, 69},

		{"female", 27, "Высшее неоконченное", "Россия", "Ленинградская область", "Одинок(а)",
			`[{"trait":"Страсть","rank":1,"score":89},{"trait":"Эмпатия","rank":2,"score":92},{"trait":"Красота","rank":3,"score":90},{"trait":"Юмор","rank":4,"score":86},{"trait":"Честность","rank":5,"score":83}]`,
			`[{"trait":"Эгоизм","rank":1,"score":38},{"trait":"Ревность","rank":2,"score":45},{"trait":"Молчаливость","rank":3,"score":30},{"trait":"Опоздания","rank":4,"score":28},{"trait":"Критика","rank":5,"score":22}]`, 77},

		{"male", 36, "Высшее оконченное", "Россия", "Самарская область", "Женат/Замужем",
			`[{"trait":"Стабильность","rank":1,"score":93},{"trait":"Надёжность","rank":2,"score":90},{"trait":"Поддержка","rank":3,"score":88},{"trait":"Интеллект","rank":4,"score":85},{"trait":"Забота","rank":5,"score":82}]`,
			`[{"trait":"Упрямство","rank":1,"score":40},{"trait":"Лень","rank":2,"score":42},{"trait":"Ревность","rank":3,"score":35},{"trait":"Забывчивость","rank":4,"score":30},{"trait":"Критика","rank":5,"score":25}]`, 75},

		{"female", 29, "Учёная степень", "Россия", "Свердловская область", "Состою в романтических отношениях",
			`[{"trait":"Мудрость","rank":1,"score":95},{"trait":"Эмпатия","rank":2,"score":93},{"trait":"Честность","rank":3,"score":91},{"trait":"Поддержка","rank":4,"score":89},{"trait":"Оптимизм","rank":5,"score":87}]`,
			`[{"trait":"Перфекционизм","rank":1,"score":28},{"trait":"Ревность","rank":2,"score":22},{"trait":"Критика","rank":3,"score":30},{"trait":"Опоздания","rank":4,"score":25},{"trait":"Эмоциональность","rank":5,"score":20}]`, 88},

		{"male", 32, "Среднее профессиональное", "Россия", "Челябинская область", "Женат/Замужем",
			`[{"trait":"Надёжность","rank":1,"score":90},{"trait":"Юмор","rank":2,"score":86},{"trait":"Забота","rank":3,"score":88},{"trait":"Стабильность","rank":4,"score":84},{"trait":"Интеллект","rank":5,"score":81}]`,
			`[{"trait":"Лень","rank":1,"score":45},{"trait":"Ревность","rank":2,"score":38},{"trait":"Забывчивость","rank":3,"score":35},{"trait":"Упрямство","rank":4,"score":32},{"trait":"Критика","rank":5,"score":28}]`, 71},

		{"female", 24, "Высшее неоконченное", "Россия", "Нижегородская область", "Одинок(а)",
			`[{"trait":"Красота","rank":1,"score":91},{"trait":"Эмпатия","rank":2,"score":88},{"trait":"Страсть","rank":3,"score":85},{"trait":"Юмор","rank":4,"score":83},{"trait":"Честность","rank":5,"score":80}]`,
			`[{"trait":"Эгоизм","rank":1,"score":40},{"trait":"Ревность","rank":2,"score":45},{"trait":"Опоздания","rank":3,"score":35},{"trait":"Молчаливость","rank":4,"score":30},{"trait":"Критика","rank":5,"score":25}]`, 73},

		{"male", 38, "Высшее оконченное", "Россия", "Воронежская область", "Женат/Замужем",
			`[{"trait":"Стабильность","rank":1,"score":92},{"trait":"Надёжность","rank":2,"score":90},{"trait":"Поддержка","rank":3,"score":87},{"trait":"Забота","rank":4,"score":85},{"trait":"Интеллект","rank":5,"score":82}]`,
			`[{"trait":"Упрямство","rank":1,"score":35},{"trait":"Лень","rank":2,"score":40},{"trait":"Ревность","rank":3,"score":30},{"trait":"Забывчивость","rank":4,"score":28},{"trait":"Критика","rank":5,"score":22}]`, 78},

		{"female", 35, "Среднее профессиональное", "Россия", "Пермский край", "Состою в романтических отношениях",
			`[{"trait":"Эмпатия","rank":1,"score":90},{"trait":"Честность","rank":2,"score":88},{"trait":"Оптимизм","rank":3,"score":85},{"trait":"Поддержка","rank":4,"score":83},{"trait":"Красота","rank":5,"score":80}]`,
			`[{"trait":"Ревность","rank":1,"score":38},{"trait":"Перфекционизм","rank":2,"score":32},{"trait":"Опоздания","rank":3,"score":35},{"trait":"Критика","rank":4,"score":28},{"trait":"Эгоизм","rank":5,"score":25}]`, 79},

		{"male", 27, "Высшее неоконченное", "Россия", "Красноярский край", "Одинок(а)",
			`[{"trait":"Юмор","rank":1,"score":89},{"trait":"Надёжность","rank":2,"score":87},{"trait":"Страсть","rank":3,"score":85},{"trait":"Забота","rank":4,"score":83},{"trait":"Интеллект","rank":5,"score":80}]`,
			`[{"trait":"Лень","rank":1,"score":48},{"trait":"Ревность","rank":2,"score":42},{"trait":"Забывчивость","rank":3,"score":38},{"trait":"Упрямство","rank":4,"score":35},{"trait":"Критика","rank":5,"score":30}]`, 66},

		{"female", 40, "Учёная степень", "Россия", "Иркутская область", "Женат/Замужем",
			`[{"trait":"Мудрость","rank":1,"score":94},{"trait":"Эмпатия","rank":2,"score":92},{"trait":"Честность","rank":3,"score":90},{"trait":"Поддержка","rank":4,"score":88},{"trait":"Оптимизм","rank":5,"score":85}]`,
			`[{"trait":"Перфекционизм","rank":1,"score":30},{"trait":"Ревность","rank":2,"score":25},{"trait":"Критика","rank":3,"score":32},{"trait":"Опоздания","rank":4,"score":28},{"trait":"Эмоциональность","rank":5,"score":22}]`, 87},

		{"male", 31, "Высшее оконченное", "Россия", "Омская область", "Состою в романтических отношениях",
			`[{"trait":"Стабильность","rank":1,"score":91},{"trait":"Надёжность","rank":2,"score":89},{"trait":"Юмор","rank":3,"score":86},{"trait":"Забота","rank":4,"score":84},{"trait":"Интеллект","rank":5,"score":81}]`,
			`[{"trait":"Упрямство","rank":1,"score":40},{"trait":"Лень","rank":2,"score":45},{"trait":"Ревность","rank":3,"score":35},{"trait":"Забывчивость","rank":4,"score":32},{"trait":"Критика","rank":5,"score":28}]`, 70},

		{"female", 23, "Высшее неоконченное", "Россия", "Волгоградская область", "Одинок(а)",
			`[{"trait":"Красота","rank":1,"score":93},{"trait":"Эмпатия","rank":2,"score":90},{"trait":"Страсть","rank":3,"score":88},{"trait":"Юмор","rank":4,"score":85},{"trait":"Честность","rank":5,"score":82}]`,
			`[{"trait":"Эгоизм","rank":1,"score":35},{"trait":"Ревность","rank":2,"score":42},{"trait":"Опоздания","rank":3,"score":30},{"trait":"Молчаливость","rank":4,"score":28},{"trait":"Критика","rank":5,"score":25}]`, 75},

		{"male", 45, "Среднее профессиональное", "Россия", "Саратовская область", "Женат/Замужем",
			`[{"trait":"Надёжность","rank":1,"score":92},{"trait":"Стабильность","rank":2,"score":90},{"trait":"Поддержка","rank":3,"score":88},{"trait":"Забота","rank":4,"score":86},{"trait":"Интеллект","rank":5,"score":83}]`,
			`[{"trait":"Лень","rank":1,"score":45},{"trait":"Ревность","rank":2,"score":38},{"trait":"Упрямство","rank":3,"score":35},{"trait":"Забывчивость","rank":4,"score":32},{"trait":"Критика","rank":5,"score":28}]`, 74},

		{"female", 34, "Высшее оконченное", "Россия", "Кемеровская область", "Состою в романтических отношениях",
			`[{"trait":"Эмпатия","rank":1,"score":90},{"trait":"Честность","rank":2,"score":88},{"trait":"Оптимизм","rank":3,"score":86},{"trait":"Поддержка","rank":4,"score":84},{"trait":"Красота","rank":5,"score":81}]`,
			`[{"trait":"Ревность","rank":1,"score":40},{"trait":"Перфекционизм","rank":2,"score":35},{"trait":"Опоздания","rank":3,"score":32},{"trait":"Критика","rank":4,"score":30},{"trait":"Эгоизм","rank":5,"score":25}]`, 78},

		{"male", 28, "Высшее неоконченное", "Россия", "Ставропольский край", "Одинок(а)",
			`[{"trait":"Юмор","rank":1,"score":88},{"trait":"Надёжность","rank":2,"score":86},{"trait":"Страсть","rank":3,"score":84},{"trait":"Забота","rank":4,"score":82},{"trait":"Интеллект","rank":5,"score":80}]`,
			`[{"trait":"Лень","rank":1,"score":50},{"trait":"Ревность","rank":2,"score":45},{"trait":"Забывчивость","rank":3,"score":40},{"trait":"Упрямство","rank":4,"score":35},{"trait":"Критика","rank":5,"score":30}]`, 65},

		{"female", 37, "Учёная степень", "Россия", "Хабаровский край", "Женат/Замужем",
			`[{"trait":"Мудрость","rank":1,"score":95},{"trait":"Эмпатия","rank":2,"score":93},{"trait":"Честность","rank":3,"score":91},{"trait":"Поддержка","rank":4,"score":89},{"trait":"Оптимизм","rank":5,"score":87}]`,
			`[{"trait":"Перфекционизм","rank":1,"score":30},{"trait":"Ревность","rank":2,"score":25},{"trait":"Критика","rank":3,"score":32},{"trait":"Опоздания","rank":4,"score":28},{"trait":"Эмоциональность","rank":5,"score":22}]`, 89},
	}

	count := 0
	for _, e := range entries {
		var profileID int
		err := db.QueryRow(`
			INSERT INTO user_profiles (gender, age, education, country_id, region_id, relationship_status, consent)
			SELECT $1, $2, $3, c.id, r.id, $4, true
			FROM countries c
			JOIN regions r ON r.country_id = c.id
			WHERE c.name = $5 AND r.name = $6
			RETURNING id
		`, e.Gender, e.Age, e.Education, e.RelationshipStatus, e.Country, e.Region).Scan(&profileID)

		if err != nil {
			http.Error(w, fmt.Sprintf("Ошибка вставки профиля для %s (%s): %v", e.Region, e.Gender, err), http.StatusInternalServerError)
			return
		}

		score, calcErr := calculateScore([]byte(e.PositiveTraits), []byte(e.NegativeTraits), e.Overall)
		if calcErr != nil {
			score = 70.0 // fallback, если расчёт сломался
		}

		// Ограничиваем score до 100
		if score > 100 {
			score = 100
		}

		_, err = db.Exec(`
			INSERT INTO test_results (profile_id, positive_traits, negative_traits, overall_satisfaction, calculated_score)
			VALUES ($1, $2, $3, $4, $5)
		`, profileID, e.PositiveTraits, e.NegativeTraits, e.Overall, score)

		if err != nil {
			http.Error(w, fmt.Sprintf("Ошибка вставки результата для %s: %v", e.Region, err), http.StatusInternalServerError)
			return
		}

		count++
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "success",
		"message":   fmt.Sprintf("Добавлено %d тестовых записей (все Россия)", count),
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
	})
}
