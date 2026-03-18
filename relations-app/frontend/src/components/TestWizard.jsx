// TestWizard.jsx
import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import axios from 'axios';
import { Button, TextField, FormControl, InputLabel, Select, MenuItem, Slider, Typography, Box } from '@mui/material';

const TestWizard = () => {
  const navigate = useNavigate();
  const [step, setStep] = useState(1);
  const [positives, setPositives] = useState(Array(5).fill(''));
  const [negatives, setNegatives] = useState(Array(5).fill(''));
  const [positiveRanks, setPositiveRanks] = useState(Array(5).fill(''));
  const [negativeRanks, setNegativeRanks] = useState(Array(5).fill(''));
  const [scores, setScores] = useState(Array(10).fill(50)); // 0-4 positives, 5-9 negatives
  const [overall, setOverall] = useState(50);

  const handleNext = () => setStep(step + 1);
  const handleBack = () => setStep(step - 1);

  const validateStep = () => {
    if (step === 1) return positives.every(t => t.trim() !== '');
    if (step === 2) return negatives.every(t => t.trim() !== '');
    if (step === 3) return new Set(positiveRanks).size === 5 && positiveRanks.every(r => r >= 1 && r <= 5);
    if (step === 4) return new Set(negativeRanks).size === 5 && negativeRanks.every(r => r >= 1 && r <= 5);
    return true;
  };

  const submitTest = () => {
    const introData = JSON.parse(localStorage.getItem('introData') || '{}');
    const positiveTraits = positives.map((trait, i) => ({ trait, rank: positiveRanks[i], score: scores[i] }));
    const negativeTraits = negatives.map((trait, i) => ({ trait, rank: negativeRanks[i], score: scores[i + 5] }));

    const data = {
      ...introData,
      positive_traits: JSON.stringify(positiveTraits),
      negative_traits: JSON.stringify(negativeTraits),
      overall_satisfaction: overall,
    };

    // Добавлено логирование для отладки
    console.log('=== ОТПРАВЛЯЕМ В ЗАПРОСЕ ===');
    console.log('introData из localStorage:', introData);
    console.log('Полный объект data:', data);
    console.log('JSON body для отправки:', JSON.stringify(data));

    localStorage.setItem('testData', JSON.stringify({
      positiveTraits,
      negativeTraits,
      overall
    }));

    axios.post('http://localhost:8080/api/submit-form', data)
      .then(res => {
        localStorage.setItem('result', JSON.stringify(res.data));
        navigate('/result');
      })
      .catch(err => {
        console.error('Submit error full:', err);
        if (err.response) {
          console.error('Server status:', err.response.status);
          console.error('Server message:', err.response.data);
        } else if (err.request) {
          console.error('No response received:', err.request);
        } else {
          console.error('Error setting up request:', err.message);
        }
      });
  };

  const getEmoji = (value) => {
    if (value > 80) return '😊'; 
    if (value >= 60 && value <= 80) return '🙂'; 
    if (value >= 40 && value <= 60) return '😐'; 
    if (value >= 20 && value <= 40) return '☹️'; 
    return '😢';
  };

  const renderStep = () => {
    switch (step) {
      case 1:
        return (
          <Box>
            <Typography variant="h6">Определение положительных качествв</Typography>
            <Typography variant="body1">Подумайте о том, что делает вашего партнера особенным и какие качества вы цените в ваших отношениях больше всего и перечислите 5 лучших качеств или сильных сторон вашего партнера.</Typography>
            {positives.map((_, i) => (
              <TextField key={i} label={`Черта ${i + 1}`} value={positives[i]} onChange={e => {
                const newPos = [...positives];
                newPos[i] = e.target.value;
                setPositives(newPos);
              }} fullWidth margin="normal" />
            ))}
          </Box>
        );
      case 2:
        return (
          <Box>
            <Typography variant="h6">Определите негативных качествв</Typography>
            <Typography variant="body1">Перечислите 5 худших качеств или проблем вашего партнера.</Typography>
            {negatives.map((_, i) => (
              <TextField key={i} label={`Черта ${i + 1}`} value={negatives[i]} onChange={e => {
                const newNeg = [...negatives];
                newNeg[i] = e.target.value;
                setNegatives(newNeg);
              }} fullWidth margin="normal" />
            ))}
          </Box>
        );
      case 3:
        return (
          <Box>
            <Typography variant="h6">Рейтинг положительных качеств</Typography>
            <Typography variant="body1">Оцените эти качества по шкале от 1 (самое важное) до 5 (наименее важное)</Typography>
            {positives.map((trait, i) => (
              <FormControl key={i} fullWidth margin="normal">
                <InputLabel>{trait || `Черта ${i + 1}`}</InputLabel>
                <Select value={positiveRanks[i]} onChange={e => {
                  const newRanks = [...positiveRanks];
                  newRanks[i] = e.target.value;
                  setPositiveRanks(newRanks);
                }}>
                  {[1, 2, 3, 4, 5].map(r => (
                    <MenuItem key={r} value={r} disabled={positiveRanks.includes(r) && positiveRanks.indexOf(r) !== i}>
                      {r}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            ))}
          </Box>
        );
      case 4:
        return (
          <Box>
            <Typography variant="h6">Рейтинг негативных качеств</Typography>
            <Typography variant="body1">Оцените эти качества по шкале от 1 (наиболее значимое) до 5 (наименее значимое)</Typography>
            {negatives.map((trait, i) => (
              <FormControl key={i} fullWidth margin="normal">
                <InputLabel>{trait || `Черта ${i + 1}`}</InputLabel>
                <Select value={negativeRanks[i]} onChange={e => {
                  const newRanks = [...negativeRanks];
                  newRanks[i] = e.target.value;
                  setNegativeRanks(newRanks);
                }}>
                  {[1, 2, 3, 4, 5].map(r => (
                    <MenuItem key={r} value={r} disabled={negativeRanks.includes(r) && negativeRanks.indexOf(r) !== i}>
                      {r}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            ))}
          </Box>
        );
      case 5:
        return (
          <Box>
            <Typography variant="h6">Оценка черт</Typography>
            <Typography variant="body1">Оцените каждую черту по шкале от 0 до 100 с учетом ваших отношений</Typography>
            {[...positives, ...negatives].map((trait, i) => (
              <Box key={i} margin="normal">
                <Typography>{trait || `Черта ${i + 1}`}</Typography>
                <Slider value={scores[i]} onChange={(_, v) => {
                  const newScores = [...scores];
                  newScores[i] = v;
                  setScores(newScores);
                }} min={0} max={100} valueLabelDisplay="auto" />
              </Box>
            ))}
          </Box>
        );
      case 6:
        return (
          <Box>
            <Typography variant="h6">Оценка общей удовлетворённости отношениями</Typography>
            <Typography variant="body1">Оцените каждую черту по шкале от 0 до 100 с учетом ваших отношений</Typography>
            <Box sx={{ display: 'flex', justifyContent: 'center', marginBottom: '20px' }}>
              <Typography variant="h2">
                {getEmoji(overall)}
              </Typography>
            </Box>
            <Slider value={overall} onChange={(_, v) => setOverall(v)} min={0} max={100} valueLabelDisplay="auto" />
          </Box>
        );
      default:
        return null;
    }
  };

  return (
    <Box maxWidth="600px" margin="auto">
      {renderStep()}
      <Box display="flex" justifyContent="space-between" marginTop="20px">
        {step > 1 && <Button onClick={handleBack} variant="outlined">Назад</Button>}
        {step < 6 ? (
          <Button onClick={handleNext} variant="contained" disabled={!validateStep()}>Далее</Button>
        ) : (
          <Button onClick={submitTest} variant="contained">Завершить</Button>
        )}
      </Box>
    </Box>
  );
};

export default TestWizard;