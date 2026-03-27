package main

import (
	"encoding/json"
	"fmt"
)

type Trait struct {
	Trait string `json:"trait"`
	Rank  int    `json:"rank"`
	Score int    `json:"score"`
}

// Функция расчета score
func calculateScore(positivesJSON, negativesJSON []byte, overall int) (float64, error) {
	var positives []Trait
	var negatives []Trait

	if err := json.Unmarshal(positivesJSON, &positives); err != nil {
		return 0, err
	}
	if err := json.Unmarshal(negativesJSON, &negatives); err != nil {
		return 0, err
	}

	weights := map[int]float64{1: 0.5, 2: 0.4, 3: 0.3, 4: 0.2, 5: 0.1}

	sumPos := 0.0
	for _, t := range positives {
		if w, ok := weights[t.Rank]; ok {
			sumPos += (float64(t.Score) / 100.0) * w
		} else {
			return 0, fmt.Errorf("invalid rank %d", t.Rank)
		}
	}

	sumNeg := 0.0
	for _, t := range negatives {
		if w, ok := weights[t.Rank]; ok {
			sumNeg += (float64(t.Score) / 100.0) * w
		} else {
			return 0, fmt.Errorf("invalid rank %d", t.Rank)
		}
	}

	// Нормализация
	normPos := sumPos / 1.5
	normNeg := sumNeg / 1.5
	if normNeg == 0 {
		normNeg = 0.0001
	}
	scom := (normPos / normNeg) * (float64(overall) / 100.0)
	score := scom * 100

	if score > 100 {
		score = 100
	}

	return scom * 100, nil
}
