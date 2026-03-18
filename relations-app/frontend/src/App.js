import React, { useState, useMemo } from 'react';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import { ThemeProvider, createTheme } from '@mui/material/styles';
import { CssBaseline, AppBar, Toolbar, Typography, Switch, Box } from '@mui/material';
import IntroForm from './components/IntroForm';
import TestWizard from './components/TestWizard';
import Result from './components/Result';
import Statistics from './components/Statistics';
import { useEffect } from 'react';

function App() {
  const [darkMode, setDarkMode] = useState(false);

  useEffect(() => {
    document.body.className = darkMode ? 'dark-theme' : 'light-theme';
  }, [darkMode]);

  const theme = useMemo(
    () =>
      createTheme({
      palette: {
        mode: darkMode ? 'dark' : 'light',
        primary: {
          main: darkMode ? '#bb86fc' : '#6200ee', // Фиолетовый для кнопок (романтично)
        },
        secondary: {
          main: darkMode ? '#03dac6' : '#03dac6', // Бирюзовый акцент
        },
        background: {
          default: darkMode ? '#121212' : '#f4f4f4', // Мягкий фон
          paper: darkMode ? '#424242' : '#ffffff',   // Для карточек/форм
        },
      },
      typography: {
        fontFamily: '"Roboto", "Helvetica", "Arial", sans-serif', // Стандарт, но можно добавить Google Fonts
        h6: {
          fontWeight: 600, // Чуть жирнее заголовки шагов
        },
      },
      components: {
        MuiButton: {
          styleOverrides: {
            root: {
              borderRadius: 8, // Мягкие углы кнопок
              textTransform: 'none',
            },
          },
        },
        MuiSlider: {
          styleOverrides: {
            root: {
              color: darkMode ? '#bb86fc' : '#6200ee', // Цвет слайдера под тему
            },
          },
        },
        MuiAppBar: {
          styleOverrides: {
            root: ({ theme }) => ({
              backgroundColor: theme.palette.mode === 'dark' ? '#3700b3' : theme.palette.primary.main, // В dark — тёмно-фиолетовый, в light — стандартный
            }),
          },
        },
      },
    }),
  [darkMode]
);

  const handleThemeChange = () => {
    setDarkMode(!darkMode);
  };

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <Router>
        <AppBar position="static">
          <Toolbar>
            <Typography variant="h6" component="div" sx={{ flexGrow: 1 }}>
              Оценка качества романтических отношений
            </Typography>
            <Switch checked={darkMode} onChange={handleThemeChange} />
            <Typography>Цветовая тема</Typography>
          </Toolbar>
        </AppBar>
        <Box style={{ padding: '20px', maxWidth: '800px', margin: 'auto' }}>
          <Routes>
            <Route path="/" element={<IntroForm />} />
            <Route path="/test" element={<TestWizard />} />
            <Route path="/result" element={<Result />} />
            <Route path="/stats" element={<Statistics />} />
          </Routes>
        </Box>
      </Router>
    </ThemeProvider>
  );
}

export default App;