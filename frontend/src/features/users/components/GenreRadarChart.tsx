import { useTheme } from '@mui/material/styles'
import { RadarChart } from '@mui/x-charts/RadarChart'
import type { GenreStat } from '@/shared/types/library'

interface GenreRadarChartProps {
  genres: GenreStat[]
  height?: number
}

export default function GenreRadarChart({ genres, height = 320 }: GenreRadarChartProps) {
  const theme = useTheme()

  const top = genres
    .slice()
    .sort((a, b) => b.PlayCount - a.PlayCount)
    .slice(0, 8)

  const metrics = top.map((g) => g.Genre)
  const data = top.map((g) => g.PlayCount)

  return (
    <RadarChart
      height={height}
      series={[
        {
          id: 'user-genres',
          label: 'Genres',
          data,
          fillArea: true,
        },
      ]}
      radar={{
        metrics,
        max: Math.max(...data, 1) * 1.2,
      }}
      colors={[theme.palette.primary.main]}
      sx={{ width: '100%' }}
    />
  )
}
