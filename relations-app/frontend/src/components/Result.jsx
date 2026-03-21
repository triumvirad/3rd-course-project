import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Typography, Button, Box } from '@mui/material';

const Result = () => {
  const navigate = useNavigate();
  const [result, setResult] = useState(null);
  const [testData, setTestData] = useState(null);

  useEffect(() => {
    const storedResult = localStorage.getItem('result');
    const storedTestData = localStorage.getItem('testData');
    if (storedResult && storedTestData) {
      setResult(JSON.parse(storedResult));
      setTestData(JSON.parse(storedTestData));
    } else {
      navigate('/');
    }
  }, [navigate]);

  if (!result || !testData) return <Typography>Загрузка...</Typography>;

  // Вес по рангу: 1 - самая важная (0.5), 5 - наименьшая (0.1)
  const getWeight = (rank) => {
    switch (rank) {
      case 1: return 0.5;
      case 2: return 0.4;
      case 3: return 0.3;
      case 4: return 0.2;
      case 5: return 0.1;
      default: return 0;
    }
  };

  // Отсортировать черты по рангу (1-5)
  const sortedPositive = [...testData.positiveTraits].sort((a, b) => a.rank - b.rank);
  const sortedNegative = [...testData.negativeTraits].sort((a, b) => a.rank - b.rank);

  // Расчёт ∑P × W
  const positiveCalcs = sortedPositive.map(trait => (trait.score / 100) * getWeight(trait.rank));
  const sumP = positiveCalcs.reduce((a, b) => a + b, 0);
  const normP = sumP / 1.5;

  // Расчёт ∑N × W
  const negativeCalcs = sortedNegative.map(trait => (trait.score / 100) * getWeight(trait.rank));
  const sumN = negativeCalcs.reduce((a, b) => a + b, 0);
  const normN = sumN / 1.5;

  // Итоговый S_COMMIT
  const bsat = testData.overall / 100;
  const scommit = Math.min(
    (normP / (normN || 1)) * bsat * 100,
    100
  ); 

  return (
    <Box textAlign="center">
      <Typography variant="h6">Показатель качества ваших отношений</Typography>
      <Typography variant="h4">{Math.min(result.score, 100).toFixed(1)}% - {result.text}</Typography>
      <Box 
        sx={{ 
          marginTop: '20px', 
          textAlign: 'left', 
          maxWidth: '600px', 
          margin: 'auto', 
          border: '2px solid', 
          borderColor: 'divider', 
          borderRadius: '8px', 
          padding: '16px' 
        }}
      >
        <Typography variant="body2" color="textSecondary" align="center" sx={{ fontStyle: 'italic' }}>
          Важно: Этот тест предназначен исключительно для развлекательных и информационных целей. Результаты основаны на ваших ответах и не являются профессиональной психологической оценкой. Они не отражают реальную ситуацию и не заменяют консультацию специалиста. Если вы испытываете серьёзные проблемы в отношениях, обратитесь к квалифицированному психологу или семейному терапевту.
        </Typography>
      </Box>
      <Box 
        sx={{ 
          marginTop: '30px', 
          textAlign: 'left', 
          maxWidth: '600px', 
          margin: 'auto', 
          border: '2px solid', 
          borderColor: 'divider', 
          borderRadius: '8px', 
          padding: '16px' 
        }}
      >
        <Typography variant="h6">Расчёт результата</Typography>
        <Typography variant="body1">
          Формула: S<sub>COMMIT</sub> = ((P × W) / 1.5) / ((N × W) / 1.5) × (B<sub>SAT</sub> / 100) × 100%
        </Typography>
        <Typography variant="subtitle1" sx={{ mt: 2 }}>Положительные черты (P × W):</Typography>
        <ul>
          {sortedPositive.map((trait, i) => (
            <li key={i}>Ранг {trait.rank}: ({trait.score}/100) × {getWeight(trait.rank)} = {positiveCalcs[i].toFixed(3)}</li>
          ))}
        </ul>
        <Typography>Сумма: {sumP.toFixed(3)}</Typography>
        <Typography>Нормализация: {sumP.toFixed(3)} / 1.5 = {normP.toFixed(3)}</Typography>
        
        <Typography variant="subtitle1" sx={{ mt: 2 }}>Негативные черты (N × W):</Typography>
        <ul>
          {sortedNegative.map((trait, i) => (
            <li key={i}>Ранг {trait.rank}: ({trait.score}/100) × {getWeight(trait.rank)} = {negativeCalcs[i].toFixed(3)}</li>
          ))}
        </ul>
        <Typography>Сумма: {sumN.toFixed(3)}</Typography>
        <Typography>Нормализация: {sumN.toFixed(3)} / 1.5 = {normN.toFixed(3)}</Typography>
        
        <Typography variant="subtitle1" sx={{ mt: 2 }}>Итоговый расчёт:</Typography>
        <Typography>S<sub>COMMIT</sub> = ({normP.toFixed(3)} / {normN.toFixed(3)}) × ({testData.overall}/100) = {(normP / (normN || 1)).toFixed(3)} × {bsat.toFixed(3)} = {scommit.toFixed(1)}%</Typography>
      </Box>
      <Button onClick={() => navigate('/stats')} variant="contained" style={{ marginTop: '20px' }}>
        Посмотреть статистику
      </Button>
    </Box>
  );
};

export default Result;