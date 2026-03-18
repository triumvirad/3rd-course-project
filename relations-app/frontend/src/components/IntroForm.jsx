// IntroForm.jsx
import React, { useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { useNavigate } from 'react-router-dom';
import axios from 'axios';
import {
  Button,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  FormControlLabel,
  Checkbox,
  Radio,
  RadioGroup,
  FormLabel,
  Typography,
} from '@mui/material';

const IntroForm = () => {
  const { register, handleSubmit, formState: { errors }, watch, setValue } = useForm();
  const navigate = useNavigate();
  const [options, setOptions] = useState({ countries: [], educations: [], statuses: [] });
  const [regions, setRegions] = useState([]);
  const selectedCountry = watch('country');

  useEffect(() => {
    axios.get('http://localhost:8080/api/start')
      .then(res => setOptions(res.data))
      .catch(err => console.error('Error loading options:', err));
  }, []);

  useEffect(() => {
    if (selectedCountry) {
      axios.get(`http://localhost:8080/api/regions?country=${encodeURIComponent(selectedCountry)}`)
        .then(res => setRegions(res.data.regions || []))
        .catch(err => console.error('Error loading regions:', err));
      setValue('region', '');
    } else {
      setRegions([]);
    }
  }, [selectedCountry, setValue]);

  const onSubmit = (data) => {
    if (!data.consent) return;
    localStorage.setItem('introData', JSON.stringify(data));
    navigate('/test');
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)} style={{ maxWidth: '400px', margin: 'auto' }}>
      <Typography variant="h5" align="center" gutterBottom>Анкета перед тестированием</Typography>
      <Typography variant="body2" color="textSecondary" align="center" gutterBottom sx={{ marginTop: '10px', fontStyle: 'italic' }}>
        Важно: Результаты теста основаны на ваших ответах и не являются профессиональной психологической оценкой. Они не отражают реальную ситуацию и не заменяют консультацию специалиста. Если вы испытываете серьёзные проблемы в отношениях, обратитесь к квалифицированному психологу или семейному терапевту.
      </Typography>

      <FormControl component="fieldset" error={!!errors.gender} fullWidth margin="normal">
        <FormLabel>Укажите Ваш пол</FormLabel>
        <RadioGroup row>
          <FormControlLabel value="male" control={<Radio />} label="Мужской" {...register('gender', { required: true })} />
          <FormControlLabel value="female" control={<Radio />} label="Женский" {...register('gender', { required: true })} />
        </RadioGroup>
        {errors.gender && <Typography color="error">Обязательное поле</Typography>}
      </FormControl>

      <TextField
        label="Введите Ваш возраст"
        type="number"
        fullWidth
        margin="normal"
        {...register('age', { required: true, min: 18, max: 100, valueAsNumber: true })}
        error={!!errors.age}
        helperText={errors.age ? 'Возраст от 18 до 100' : ''}
      />

      <FormControl fullWidth margin="normal" error={!!errors.education}>
        <InputLabel>Уровень образования</InputLabel>
        <Select {...register('education', { required: true })}>
          {options.educations.map((edu, i) => <MenuItem key={i} value={edu}>{edu}</MenuItem>)}
        </Select>
        {errors.education && <Typography color="error">Обязательное поле</Typography>}
      </FormControl>

      <FormControl fullWidth margin="normal" error={!!errors.country}>
        <InputLabel>Страна</InputLabel>
        <Select {...register('country', { required: true })}>
          {options.countries.map((country, i) => <MenuItem key={i} value={country}>{country}</MenuItem>)}
        </Select>
        {errors.country && <Typography color="error">Обязательное поле</Typography>}
      </FormControl>

      <FormControl fullWidth margin="normal" error={!!errors.region}>
        <InputLabel>Регион</InputLabel>
        <Select {...register('region', { required: false })} disabled={regions.length === 0}>
          <MenuItem value="">Не указан</MenuItem>
          {regions.map((reg, i) => <MenuItem key={i} value={reg}>{reg}</MenuItem>)}
        </Select>
        {regions.length === 0 && selectedCountry && <Typography color="info">Регионы загружаются...</Typography>}
      </FormControl>

      <FormControl fullWidth margin="normal" error={!!errors.relationship_status}>
        <InputLabel>Семейное положение</InputLabel>
        <Select {...register('relationship_status', { required: true })}>
          {options.statuses.map((status, i) => <MenuItem key={i} value={status}>{status}</MenuItem>)}
        </Select>
        {errors.relationship_status && <Typography color="error">Обязательное поле</Typography>}
      </FormControl>

      <FormControlLabel
        control={<Checkbox {...register('consent', { required: true })} />}
        label="Согласен на обработку данных для исследования с соблюдением конфиденциальности"
      />
      {errors.consent && <Typography color="error">Необходимо согласие</Typography>}

      <Button type="submit" variant="contained" color="primary" fullWidth style={{ marginTop: '20px' }}>
        Начать тест
      </Button>
    </form>
  );
};

export default IntroForm;