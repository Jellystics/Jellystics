import { useState, useMemo } from 'react'
import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  flexRender,
  type ColumnDef,
  type SortingState,
} from '@tanstack/react-table'
import {
  Box,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TablePagination,
  TextField,
  InputAdornment,
  IconButton,
  Popover,
  Checkbox,
  FormControlLabel,
  Skeleton,
  Typography,
  Stack,
  Tooltip,
  Badge,
  Button,
  Chip,
  FormControl,
  Select,
  MenuItem,
  ListItemText,
  Divider,
} from '@mui/material'
import {
  Search20Regular,
  ColumnTriple24Regular,
  ArrowUp20Regular,
  ArrowDown20Regular,
  ArrowSync24Regular,
  Filter24Regular,
} from '@fluentui/react-icons'
import { useTranslation } from 'react-i18next'
import { useDebounce } from '@/shared/hooks/useDebounce'

export interface FilterDef {
  id: string
  label: string
  type: 'select' | 'range' | 'daterange'
  unit?: string
  transform?: (raw: number) => number
  /** If true, this filter is handled server-side — client filtering is skipped and onServerFilterChange is called on Apply/Clear */
  serverSide?: boolean
}

type SelectFilter = string[]
type RangeFilter = [number | undefined, number | undefined]
type DateRangeFilter = [string | undefined, string | undefined]
type FilterState = Record<string, SelectFilter | RangeFilter | DateRangeFilter>

interface DataTableProps<T> {
  data: T[]
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  columns: ColumnDef<T, any>[]
  loading?: boolean
  searchable?: boolean
  searchPlaceholder?: string
  onRefresh?: () => void
  initialColumnVisibility?: Record<string, boolean>
  filterDefs?: FilterDef[]
  /** Called when a server-side filter is applied or cleared; receives only the active server-side filter values */
  onServerFilterChange?: (serverFilters: Record<string, SelectFilter | RangeFilter | DateRangeFilter>) => void
}

export default function DataTable<T>({
  data,
  columns,
  loading,
  searchable = true,
  searchPlaceholder,
  onRefresh,
  initialColumnVisibility,
  filterDefs,
  onServerFilterChange,
}: DataTableProps<T>) {
  const { t } = useTranslation()
  const [sorting, setSorting] = useState<SortingState>([])
  const [globalFilter, setGlobalFilter] = useState('')
  const [columnVisibility, setColumnVisibility] = useState<Record<string, boolean>>(initialColumnVisibility ?? {})
  const [colAnchor, setColAnchor] = useState<null | HTMLElement>(null)
  const [filterAnchor, setFilterAnchor] = useState<null | HTMLElement>(null)
  const [activeFilters, setActiveFilters] = useState<FilterState>({})
  const [pendingFilters, setPendingFilters] = useState<FilterState>({})

  const debouncedFilter = useDebounce(globalFilter, 300)

  // Unique values for select filters from raw data
  const uniqueValues = useMemo(() => {
    const result: Record<string, string[]> = {}
    filterDefs?.forEach((def) => {
      if (def.type !== 'select') return
      const set = new Set<string>()
      data.forEach((row) => {
        const val = String((row as Record<string, unknown>)[def.id] ?? '')
        if (val && val !== 'undefined' && val !== 'null') set.add(val)
      })
      result[def.id] = Array.from(set).sort()
    })
    return result
  }, [data, filterDefs])

  // Apply filters client-side
  const filteredData = useMemo(() => {
    if (!filterDefs?.length) return data
    const hasActive = filterDefs.some((def) => {
      const f = activeFilters[def.id]
      if (!f) return false
      if (def.type === 'select') return (f as SelectFilter).length > 0
      const [a, b] = f as RangeFilter | DateRangeFilter
      return a !== undefined || b !== undefined
    })
    if (!hasActive) return data

    return data.filter((row) =>
      filterDefs.every((def) => {
        if (def.serverSide) return true  // handled by API
        const f = activeFilters[def.id]
        if (!f) return true

        if (def.type === 'select') {
          const selected = f as SelectFilter
          if (!selected.length) return true
          const val = String((row as Record<string, unknown>)[def.id] ?? '')
          return selected.includes(val)
        }

        if (def.type === 'range') {
          const [min, max] = f as RangeFilter
          if (min === undefined && max === undefined) return true
          const raw = ((row as Record<string, unknown>)[def.id] as number) ?? 0
          const val = def.transform ? def.transform(raw) : raw
          if (min !== undefined && val < min) return false
          if (max !== undefined && val > max) return false
        }

        if (def.type === 'daterange') {
          const [from, to] = f as DateRangeFilter
          if (from === undefined && to === undefined) return true
          const rawDate = String((row as Record<string, unknown>)[def.id] ?? '').substring(0, 10)
          if (from !== undefined && rawDate < from) return false
          if (to !== undefined && rawDate > to) return false
        }

        return true
      })
    )
  }, [data, activeFilters, filterDefs])

  const filterCount = useMemo(() => {
    if (!filterDefs) return 0
    return filterDefs.filter((def) => {
      const f = activeFilters[def.id]
      if (!f) return false
      if (def.type === 'select') return (f as SelectFilter).length > 0
      const [a, b] = f as RangeFilter | DateRangeFilter
      return a !== undefined || b !== undefined
    }).length
  }, [activeFilters, filterDefs])

  const table = useReactTable({
    data: filteredData,
    columns,
    state: { sorting, globalFilter: debouncedFilter, columnVisibility },
    onSortingChange: setSorting,
    onGlobalFilterChange: setGlobalFilter,
    onColumnVisibilityChange: setColumnVisibility,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    initialState: { pagination: { pageSize: 25 } },
  })

  const skeletonRows = useMemo(() => Array.from({ length: 10 }), [])

  const notifyServerFilters = (filters: FilterState) => {
    if (!onServerFilterChange || !filterDefs) return
    const serverFilters: Record<string, SelectFilter | RangeFilter | DateRangeFilter> = {}
    filterDefs.forEach((def) => {
      if (def.serverSide && filters[def.id]) {
        serverFilters[def.id] = filters[def.id] as SelectFilter | RangeFilter | DateRangeFilter
      }
    })
    onServerFilterChange(serverFilters)
  }

  const clearFilter = (defId: string, selectValue?: string) => {
    const current = activeFilters[defId]
    if (!current) return
    let newFilters: FilterState
    if (selectValue !== undefined) {
      const newVal = (current as SelectFilter).filter((v) => v !== selectValue)
      if (!newVal.length) { const n = { ...activeFilters }; delete n[defId]; newFilters = n }
      else newFilters = { ...activeFilters, [defId]: newVal }
    } else {
      const n = { ...activeFilters }; delete n[defId]; newFilters = n
    }
    setActiveFilters(newFilters)
    notifyServerFilters(newFilters)
  }

  return (
    <Box>
      {/* Toolbar */}
      <Stack direction="row" spacing={1} sx={{ mb: filterCount > 0 ? 1 : 2, alignItems: 'center' }}>
        {searchable && (
          <TextField
            size="small"
            placeholder={searchPlaceholder ?? t('common.search')}
            value={globalFilter}
            onChange={(e) => setGlobalFilter(e.target.value)}
            slotProps={{
              input: {
                startAdornment: (
                  <InputAdornment position="start">
                    <Search20Regular style={{ fontSize: 16, color: 'var(--mui-palette-text-secondary)' }} />
                  </InputAdornment>
                ),
              },
            }}
            sx={{ width: 240 }}
          />
        )}
        <Box sx={{ flexGrow: 1 }} />
        {onRefresh && (
          <Tooltip title={t('common.refresh')}>
            <span>
              <IconButton size="small" onClick={onRefresh} sx={{ color: 'text.secondary' }} disabled={loading}>
                <ArrowSync24Regular style={{ fontSize: 20 }} />
              </IconButton>
            </span>
          </Tooltip>
        )}
        {filterDefs?.length ? (
          <Tooltip title={t('common.filters', 'Filtres')}>
            <Badge badgeContent={filterCount} color="primary" invisible={filterCount === 0}>
              <IconButton
                size="small"
                onClick={(e) => { setPendingFilters(activeFilters); setFilterAnchor(e.currentTarget) }}
                sx={{ color: filterCount > 0 ? 'primary.main' : 'text.secondary' }}
              >
                <Filter24Regular style={{ fontSize: 20 }} />
              </IconButton>
            </Badge>
          </Tooltip>
        ) : null}
        <Tooltip title={t('common.columns')}>
          <IconButton
            size="small"
            onClick={(e) => setColAnchor(e.currentTarget)}
            sx={{ color: 'text.secondary' }}
          >
            <ColumnTriple24Regular style={{ fontSize: 20 }} />
          </IconButton>
        </Tooltip>

        {/* Column visibility popover */}
        <Popover
          open={Boolean(colAnchor)}
          anchorEl={colAnchor}
          onClose={() => setColAnchor(null)}
          anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
          transformOrigin={{ vertical: 'top', horizontal: 'right' }}
          slotProps={{ paper: { sx: { p: 1.5, minWidth: 180 } } }}
        >
          <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1 }}>
            {t('common.columns')}
          </Typography>
          {table.getAllLeafColumns().map((col) => (
            <Box key={col.id}>
              <FormControlLabel
                control={
                  <Checkbox
                    size="small"
                    checked={col.getIsVisible()}
                    onChange={col.getToggleVisibilityHandler()}
                  />
                }
                label={<Typography variant="body2">{String(col.columnDef.header ?? col.id)}</Typography>}
              />
            </Box>
          ))}
        </Popover>

        {/* Filter popover */}
        {filterDefs?.length ? (
          <Popover
            open={Boolean(filterAnchor)}
            anchorEl={filterAnchor}
            onClose={() => setFilterAnchor(null)}
            anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
            transformOrigin={{ vertical: 'top', horizontal: 'right' }}
            slotProps={{ paper: { sx: { width: 300 } } }}
          >
            <Box sx={{ p: 2, pb: 1 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 1.5 }}>
                <Typography variant="subtitle2" sx={{ fontWeight: 600, flex: 1 }}>
                  {t('common.filters', 'Filtres')}
                </Typography>
                <Button
                  size="small"
                  onClick={() => setPendingFilters({})}
                  disabled={!Object.values(pendingFilters).some((v) => Array.isArray(v) && v.some((x) => x !== undefined && (x as unknown) !== ''))}
                >
                  {t('common.clearAll', 'Tout effacer')}
                </Button>
              </Box>
              <Divider sx={{ mb: 2 }} />

              <Stack spacing={2}>
                {filterDefs.map((def) => (
                  <Box key={def.id}>
                    <Typography
                      variant="caption"
                      sx={{ fontWeight: 700, color: 'text.secondary', textTransform: 'uppercase', letterSpacing: 0.5, display: 'block', mb: 0.75 }}
                    >
                      {def.label}
                    </Typography>

                    {def.type === 'select' && (
                      <FormControl size="small" fullWidth>
                        <Select
                          multiple
                          displayEmpty
                          value={(pendingFilters[def.id] as SelectFilter) ?? []}
                          onChange={(e) =>
                            setPendingFilters((prev) => ({
                              ...prev,
                              [def.id]: e.target.value as string[],
                            }))
                          }
                          renderValue={(selected) => {
                            const s = selected as string[]
                            if (!s.length) return <Typography variant="body2" color="text.secondary">{t('common.all', 'Tous')}</Typography>
                            if (s.length === 1) return <Typography variant="body2">{s[0]}</Typography>
                            return <Typography variant="body2">{s.length} {t('common.selected', 'sélectionnés')}</Typography>
                          }}
                        >
                          {(uniqueValues[def.id] ?? []).map((val) => (
                            <MenuItem key={val} value={val} dense>
                              <Checkbox
                                size="small"
                                checked={((pendingFilters[def.id] as SelectFilter) ?? []).includes(val)}
                                sx={{ py: 0, pl: 0, pr: 0.5 }}
                              />
                              <ListItemText
                                primary={val}
                                slotProps={{ primary: { style: { fontSize: 13 } } }}
                              />
                            </MenuItem>
                          ))}
                        </Select>
                      </FormControl>
                    )}

                    {def.type === 'range' && (
                      <Box sx={{ display: 'flex', gap: 1 }}>
                        <TextField
                          size="small"
                          type="number"
                          placeholder={`Min${def.unit ? ` (${def.unit})` : ''}`}
                          value={(pendingFilters[def.id] as RangeFilter)?.[0] ?? ''}
                          onChange={(e) =>
                            setPendingFilters((prev) => ({
                              ...prev,
                              [def.id]: [
                                e.target.value !== '' ? Number(e.target.value) : undefined,
                                (prev[def.id] as RangeFilter)?.[1],
                              ],
                            }))
                          }
                          sx={{ flex: 1 }}
                        />
                        <TextField
                          size="small"
                          type="number"
                          placeholder={`Max${def.unit ? ` (${def.unit})` : ''}`}
                          value={(pendingFilters[def.id] as RangeFilter)?.[1] ?? ''}
                          onChange={(e) =>
                            setPendingFilters((prev) => ({
                              ...prev,
                              [def.id]: [
                                (prev[def.id] as RangeFilter)?.[0],
                                e.target.value !== '' ? Number(e.target.value) : undefined,
                              ],
                            }))
                          }
                          sx={{ flex: 1 }}
                        />
                      </Box>
                    )}

                    {def.type === 'daterange' && (
                      <Stack spacing={1}>
                        <TextField
                          size="small"
                          type="date"
                          label={t('timeRange.from')}
                          value={(pendingFilters[def.id] as DateRangeFilter)?.[0] ?? ''}
                          onChange={(e) =>
                            setPendingFilters((prev) => ({
                              ...prev,
                              [def.id]: [
                                e.target.value !== '' ? e.target.value : undefined,
                                (prev[def.id] as DateRangeFilter)?.[1],
                              ],
                            }))
                          }
                          slotProps={{ inputLabel: { shrink: true } }}
                          fullWidth
                        />
                        <TextField
                          size="small"
                          type="date"
                          label={t('timeRange.to')}
                          value={(pendingFilters[def.id] as DateRangeFilter)?.[1] ?? ''}
                          onChange={(e) =>
                            setPendingFilters((prev) => ({
                              ...prev,
                              [def.id]: [
                                (prev[def.id] as DateRangeFilter)?.[0],
                                e.target.value !== '' ? e.target.value : undefined,
                              ],
                            }))
                          }
                          slotProps={{ inputLabel: { shrink: true } }}
                          fullWidth
                        />
                      </Stack>
                    )}
                  </Box>
                ))}
              </Stack>
            </Box>

            {/* Footer buttons */}
            <Box sx={{ display: 'flex', gap: 1, p: 1.5, pt: 1, borderTop: '1px solid', borderColor: 'divider' }}>
              <Button
                fullWidth
                variant="outlined"
                size="small"
                onClick={() => setPendingFilters({})}
              >
                {t('common.reset', 'Réinitialiser')}
              </Button>
              <Button
                fullWidth
                variant="contained"
                size="small"
                onClick={() => { setActiveFilters(pendingFilters); notifyServerFilters(pendingFilters); setFilterAnchor(null) }}
              >
                {t('common.apply', 'Appliquer')}
              </Button>
            </Box>
          </Popover>
        ) : null}
      </Stack>

      {/* Active filter chips */}
      {filterCount > 0 && filterDefs && (
        <Box sx={{ display: 'flex', gap: 0.75, flexWrap: 'wrap', mb: 1.5, alignItems: 'center' }}>
          {filterDefs.flatMap((def) => {
            const f = activeFilters[def.id]
            if (!f) return []

            if (def.type === 'select') {
              const selected = f as SelectFilter
              return selected.map((val) => (
                <Chip
                  key={`${def.id}-${val}`}
                  label={`${def.label}: ${val}`}
                  size="small"
                  onDelete={() => clearFilter(def.id, val)}
                />
              ))
            }

            if (def.type === 'range') {
              const [min, max] = f as RangeFilter
              if (min === undefined && max === undefined) return []
              const label =
                min !== undefined && max !== undefined
                  ? `${def.label}: ${min}–${max}${def.unit ? ` ${def.unit}` : ''}`
                  : min !== undefined
                    ? `${def.label}: ≥${min}${def.unit ? ` ${def.unit}` : ''}`
                    : `${def.label}: ≤${max}${def.unit ? ` ${def.unit}` : ''}`
              return [<Chip key={def.id} label={label} size="small" onDelete={() => clearFilter(def.id)} />]
            }

            if (def.type === 'daterange') {
              const [from, to] = f as DateRangeFilter
              if (from === undefined && to === undefined) return []
              const label =
                from !== undefined && to !== undefined
                  ? `${def.label}: ${from} – ${to}`
                  : from !== undefined
                    ? `${def.label}: ≥ ${from}`
                    : `${def.label}: ≤ ${to}`
              return [<Chip key={def.id} label={label} size="small" onDelete={() => clearFilter(def.id)} />]
            }

            return []
          })}
          <Button size="small" onClick={() => { setActiveFilters({}); notifyServerFilters({}) }} sx={{ fontSize: 12 }}>
            {t('common.clearAll', 'Tout effacer')}
          </Button>
        </Box>
      )}

      {/* Table */}
      <Box sx={{ overflow: 'hidden', borderRadius: 1, border: '1px solid', borderColor: 'divider' }}>
        <TableContainer sx={{ boxShadow: 'none', overflowX: 'auto', scrollbarGutter: 'stable' }}>
          <Table size="small" stickyHeader sx={{ minWidth: 600 }}>
            <TableHead>
              {table.getHeaderGroups().map((hg) => (
                <TableRow key={hg.id}>
                  {hg.headers.map((header) => {
                    const sorted = header.column.getIsSorted()
                    return (
                      <TableCell
                        key={header.id}
                        sx={{
                          fontWeight: 600,
                          fontSize: 12,
                          color: 'text.secondary',
                          cursor: header.column.getCanSort() ? 'pointer' : 'default',
                          userSelect: 'none',
                          whiteSpace: 'nowrap',
                          bgcolor: 'background.paper',
                        }}
                        onClick={header.column.getToggleSortingHandler()}
                      >
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                          {flexRender(header.column.columnDef.header, header.getContext())}
                          {sorted === 'asc' && <ArrowUp20Regular style={{ fontSize: 14 }} />}
                          {sorted === 'desc' && <ArrowDown20Regular style={{ fontSize: 14 }} />}
                        </Box>
                      </TableCell>
                    )
                  })}
                </TableRow>
              ))}
            </TableHead>
            <TableBody>
              {loading
                ? skeletonRows.map((_, i) => (
                    <TableRow key={i} sx={{ height: 43 }}>
                      {columns.map((_, j) => (
                        <TableCell key={j}>
                          <Skeleton variant="text" sx={{ fontSize: '0.875rem' }} />
                        </TableCell>
                      ))}
                    </TableRow>
                  ))
                : table.getRowModel().rows.map((row) => (
                    <TableRow
                      key={row.id}
                      hover
                      sx={{ height: 43, cursor: 'pointer', '&:last-child td': { border: 0 } }}
                    >
                      {row.getVisibleCells().map((cell) => (
                        <TableCell key={cell.id} sx={{ fontSize: 13 }}>
                          {flexRender(cell.column.columnDef.cell, cell.getContext())}
                        </TableCell>
                      ))}
                    </TableRow>
                  ))}
            </TableBody>
          </Table>
        </TableContainer>
      </Box>

      <Box sx={{ mt: 1 }}>
        <TablePagination
          component="div"
          count={table.getFilteredRowModel().rows.length}
          page={table.getState().pagination.pageIndex}
          rowsPerPage={table.getState().pagination.pageSize}
          rowsPerPageOptions={[10, 25, 50, 100]}
          onPageChange={(_, p) => table.setPageIndex(p)}
          onRowsPerPageChange={(e) => table.setPageSize(Number(e.target.value))}
        />
      </Box>
    </Box>
  )
}
