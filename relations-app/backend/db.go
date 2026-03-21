package main

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

func createDb(db *sql.DB) error {
	// Страны
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS countries (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) UNIQUE NOT NULL
		);
	`)
	if err != nil {
		return err
	}

	// Инициализация стран
	_, err = db.Exec(`
		INSERT INTO countries (name) VALUES
		('Россия'), ('Беларусь'), ('Казахстан'), ('Армения'), ('Таджикистан'), ('Туркменистан'), ('Киргизия'), ('Молдавия'), ('Украина')
		ON CONFLICT DO NOTHING;
	`)
	if err != nil {
		return err
	}

	// Регионы
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS regions (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			country_id INTEGER REFERENCES countries(id) NOT NULL,
			UNIQUE (name, country_id)
		);
	`)
	if err != nil {
		return err
	}

	// Инициализация регионов
	_, err = db.Exec(`
		INSERT INTO regions (name, country_id) VALUES
		-- Россия (country_id = 1)
		('Москва', 1), ('Санкт-Петербург', 1), ('Московская область', 1),
		('Ленинградская область', 1), ('Новосибирская область', 1),
		('Екатеринбург', 1), ('Краснодарский край', 1), ('Ростовская область', 1),
		('Татарстан', 1), ('Башкортостан', 1), ('Пермский край', 1),
		
		-- Беларусь (country_id = 2)
		('Минск', 2), ('Брестская область', 2), ('Гомельская область', 2),
		('Гродненская область', 2), ('Могилёвская область', 2), ('Витебская область', 2),

		-- Казахстан (country_id = 3)
		('Абайская область', 3), ('Акмолинская область', 3), ('Актюбинская область', 3),
		('Алматинская область', 3), ('Атырауская область', 3), ('Восточно-Казахстанская область', 3),
		('Жамбылская область', 3), ('Жетысуская область', 3), ('Карагандинская область', 3),
		('Костанайская область', 3), ('Кызылординская область', 3), ('Мангистауская область', 3),
		('Северо-Казахстанская область', 3), ('Павлодарская область', 3), ('Туркестанская область', 3),
		('Улытауская область', 3), ('Западно-Казахстанская область', 3),
		('Астана', 3), ('Алматы', 3), ('Шымкент', 3),
		
		-- Армения (country_id = 4)
		('Арагацотн', 4), ('Арарат', 4), ('Армавир', 4),
		('Гегаркуник', 4), ('Котайк', 4), ('Лори', 4),
		('Ширак', 4), ('Сюник', 4), ('Тавуш', 4), ('Вайоц Дзор', 4),
		('Ереван', 4),
		
		-- Таджикистан (country_id = 5)
		('Согдийская область', 5), ('Хатлонская область', 5), ('Горно-Бадахшанская автономная область', 5),
		('Районы республиканского подчинения', 5), ('Душанбе', 5),
		
		-- Туркменистан (country_id = 6)
		('Ахалский велаят', 6), ('Балканский велаят', 6), ('Дашогузский велаят', 6),
		('Лебапский велаят', 6), ('Марыйский велаят', 6), ('Ашхабад', 6),
		
		-- Киргизия (country_id = 7)
		('Баткенская область', 7), ('Чуйская область', 7), ('Джалал-Абадская область', 7),
		('Иссык-Кульская область', 7), ('Нарынская область', 7), ('Ошская область', 7),
		('Таласская область', 7), ('Бишкек', 7), ('Ош', 7),
		
	`)
	if err != nil {
		return err
	}

	// user_profiles
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_profiles (
			id SERIAL PRIMARY KEY,
			gender VARCHAR(10) NOT NULL,
			age INTEGER NOT NULL CHECK (age >= 18 AND age <= 100),
			education VARCHAR(50) NOT NULL,
			country_id INTEGER REFERENCES countries(id) NOT NULL,
			region_id INTEGER REFERENCES regions(id) NOT NULL,
			relationship_status VARCHAR(50) NOT NULL,
			consent BOOLEAN NOT NULL DEFAULT TRUE
		);
	`)
	if err != nil {
		return err
	}

	// test_results
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS test_results (
			id SERIAL PRIMARY KEY,
			profile_id INTEGER REFERENCES user_profiles(id) NOT NULL,
			positive_traits JSONB NOT NULL,
			negative_traits JSONB NOT NULL,
			overall_satisfaction INTEGER NOT NULL CHECK (overall_satisfaction BETWEEN 0 AND 100),
			calculated_score FLOAT NOT NULL
		);
	`)
	if err != nil {
		return err
	}
	log.Println("Tables created or already exist")
	return nil
}
