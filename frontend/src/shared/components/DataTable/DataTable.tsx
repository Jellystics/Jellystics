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
  Paper,
  Stack,
} from '@mui/material'
import {
  Search20Regular,
  ColumnTriple24Regular,
  ArrowUp20Regular,
  ArrowDown20Regular,
} from '@fluentui/react-icons'
import { useTranslation } from 'react-i18next'
import { useDebounce } from '@/shared/hooks/useDebounce'

interface DataTableProps<T> {
  data: T[]
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  columns: ColumnDef<T, any>[]
  loading?: boolean
  searchable?: boolean
  searchPlaceholder?: string
}

export default function DataTable<T>({
  data,
  columns,
  loading,
  searchable = true,
  searchPlaceholder,
}: DataTableProps<T>) {
  const { t } = useTranslation()
  const [sorting, setSorting] = useState<SortingState>([])
  const [globalFilter, setGlobalFilter] = useState('')
  const [columnVisibility, setColumnVisibility] = useState<Record<string, boolean>>({})
  const [colAnchor, setColAnchor] = useState<null | HTMLElement>(null)

  const debouncedFilter = useDebounce(globalFilter, 300)

  const table = useReactTable({
    data,
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

  return (
    <Box>
      {/* Toolbar */}
      <Stack direction="row" spacing={1} sx={{ mb: 2, alignItems: 'center' }}>
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
        <IconButton
          size="small"
          onClick={(e) => setColAnchor(e.currentTarget)}
          sx={{ color: 'text.secondary' }}
        >
          <ColumnTriple24Regular style={{ fontSize: 20 }} />
        </IconButton>
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
      </Stack>

      {/* Table */}
      <TableContainer
        component={Paper}
        sx={{
          boxShadow: 'none',
          border: '1px solid',
          borderColor: 'divider',
          overflowX: 'auto',
        }}
      >
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
