import React, { useEffect, useState } from 'react';
import axios from 'axios';
import { Typography, Box } from '@mui/material';

const ComparisonDiagram = ({ center, items }) => {
  if (!center || items.length === 0) {
    return <Typography>Недостаточно данных для диаграммы.</Typography>;
  }

  const width = 600;
  const height = 600;
  const cx = width / 2;
  const cy = height / 2;
  const centerRadius = 50;
  const itemRadius = 40;
  const orbitRadius = 200 + (items.length > 10 ? (items.length - 10) * 10 : 0);

  return (
    <>
      <svg width={width} height={height}>
        <defs>
          <marker id="arrowhead" markerWidth="10" markerHeight="7" refX="10" refY="3.5" orient="auto" markerUnits="strokeWidth">
            <polygon points="0 0, 10 3.5, 0 7" fill="currentColor" />
          </marker>
        </defs>

        <circle cx={cx} cy={cy} r={centerRadius} fill="lightblue" />
        <text x={cx} y={cy} textAnchor="middle" dy=".3em" fontSize="14">{center.label}</text>

        {/* Элементы вокруг */}
        {items.map((item, i) => {
          const angle = (2 * Math.PI / items.length) * i;
          const x = cx + orbitRadius * Math.cos(angle);
          const y = cy + orbitRadius * Math.sin(angle);
          item.x = x;
          item.y = y;
          return (
            <g key={i}>
              <circle cx={x} cy={y} r={itemRadius} fill="lightgray" />
              <text x={x} y={y} textAnchor="middle" dy=".3em" fontSize="14">{item.label}</text>
            </g>
          );
        })}

        {/* Стрелки */}
        {items.map((item, i) => {
          const diff = center.avg - item.avg;
          if (diff === 0) {
            const dx = item.x - cx;
            const dy = item.y - cy;
            const dist = Math.sqrt(dx * dx + dy * dy);
            const ux = dx / dist;
            const uy = dy / dist;
            const startX = cx + ux * centerRadius;
            const startY = cy + uy * centerRadius;
            const endX = item.x - ux * itemRadius;
            const endY = item.y - uy * itemRadius;
            return (
              <line
                key={i}
                x1={startX}
                y1={startY}
                x2={endX}
                y2={endY}
                stroke="currentColor"
                strokeDasharray="5,5"
                style={{ color: 'gray' }}
              />
            );
          } else {
            const color = diff > 0 ? 'green' : 'red';
            const from = diff > 0 ? { x: cx, y: cy, r: centerRadius } : { x: item.x, y: item.y, r: itemRadius };
            const to = diff > 0 ? { x: item.x, y: item.y, r: itemRadius } : { x: cx, y: cy, r: centerRadius };
            const dx = to.x - from.x;
            const dy = to.y - from.y;
            const dist = Math.sqrt(dx * dx + dy * dy);
            const ux = dx / dist;
            const uy = dy / dist;
            const startX = from.x + ux * from.r;
            const startY = from.y + uy * from.r;
            const endX = to.x - ux * to.r;
            const endY = to.y - uy * to.r;
            return (
              <line
                key={i}
                x1={startX}
                y1={startY}
                x2={endX}
                y2={endY}
                stroke="currentColor"
                strokeWidth="2"
                markerEnd="url(#arrowhead)"
                style={{ color: color }}
              />
            );
          }
        })}
      </svg>
      <Box mt={2}>
        <Typography variant="body2">
          Зелёная стрелочка от центрального показателя к другому означает, что качество отношений по центральному показателю лучше.
          Красная стрелочка от другого к центральному означает, что по другому показателю лучше.
        </Typography>
      </Box>
    </>
  );
};

const Statistics = () => {
  const [stats, setStats] = useState({});
  const [userCountry, setUserCountry] = useState('');
  const [userRegion, setUserRegion] = useState('');
  const [userEducation, setUserEducation] = useState('');
  const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';
  useEffect(() => {
    const introData = JSON.parse(localStorage.getItem('introData') || '{}');
    setUserCountry(introData.country || 'Неизвестно');
    setUserRegion(introData.region || 'Не указан');
    setUserEducation(introData.education || 'Неизвестно');

    axios.get(`${API_URL}/api/statistics`)
      .then(res => setStats(res.data))
      .catch(err => console.error('Stats error:', err));
  }, [API_URL]);

  const generateComparison = (group, items) => {
    if (!items || Object.keys(items).length < 2) return ['Недостаточно данных для сравнения.'];
    const sorted = Object.entries(items).sort((a, b) => b[1] - a[1]);
    return sorted.map((item, index) => {
      if (index < sorted.length - 1) {
        let labelA = item[0];
        let labelB = sorted[index + 1][0];
        if (group === 'gender') {
          labelA = labelA === 'male' ? 'мужчин' : 'женщин';
          labelB = labelB === 'male' ? 'мужчин' : 'женщин';
        }
        return `Качество отношений у ${labelA} выше, чем у ${labelB}.`;
      }
      return null;
    }).filter(Boolean);
  };

  return (
    <Box>
      <Typography variant="h5">Статистика </Typography>
      <Typography>Все относительно, это не замена профессиональной консультации.</Typography>
      
      <Typography variant="h6" style={{ marginTop: '20px' }}>По полу</Typography>
      {generateComparison('gender', stats.gender || {}).map((phrase, i) => (
        <Typography key={i}>{phrase}</Typography>
      ))}

      <Typography variant="h6" style={{ marginTop: '20px' }}>По уровню образования</Typography>
      {userEducation !== 'Неизвестно' && stats.education && stats.education[userEducation] ? (
        <ComparisonDiagram
          center={{ label: userEducation, avg: stats.education[userEducation] }}
          items={Object.entries(stats.education)
            .filter(([k]) => k !== userEducation)
            .map(([k, v]) => ({ label: k, avg: v }))}
        />
      ) : (
        <Typography>Уровень образования не указан или недостаточно данных.</Typography>
      )}

      <Typography variant="h6" style={{ marginTop: '20px' }}>По стране</Typography>
      {userCountry !== 'Неизвестно' && stats.country && stats.country[userCountry] !== undefined ? (
        <ComparisonDiagram
          center={{ label: userCountry, avg: stats.country[userCountry] }}
          items={Object.entries(stats.country)
            .filter(([k]) => k !== userCountry)
            .map(([k, v]) => ({ label: k, avg: v }))}
        />
      ) : (
        <Typography>Страна не указана или недостаточно данных.</Typography>
      )}

      <Typography variant="h6" style={{ marginTop: '20px' }}>По региону</Typography>
      {userRegion !== 'Не указан' && stats.regionsByCountry && stats.regionsByCountry[userCountry] && stats.regionsByCountry[userCountry][userRegion] !== undefined ? (
        <ComparisonDiagram
          center={{ label: userRegion, avg: stats.regionsByCountry[userCountry][userRegion] }}
          items={Object.entries(stats.regionsByCountry[userCountry])
            .filter(([k]) => k !== userRegion)
            .map(([k, v]) => ({ label: k, avg: v }))}
        />
      ) : (
        <Typography>Регион не указан или недостаточно данных для сравнения.</Typography>
      )}
    </Box>
  );
};

export default Statistics;