import {
  Backdrop,
  Button,
  CircularProgress,
  Container,
  Tooltip,
  Typography,
} from '@material-ui/core'
import { DataGrid } from '@material-ui/data-grid'
import { DirectionsBoat } from '@material-ui/icons'
import { JsonParam, StringParam, useQueryParam } from 'use-query-params'
import { Link } from 'react-router-dom'
import { pathForExactJob, relativeTime } from '../helpers'
import Alert from '@material-ui/lab/Alert'
import GridToolbar from '../datagrid/GridToolbar'
import PropTypes from 'prop-types'
import React, { Fragment, useEffect } from 'react'

/**
 * JobRunsTable shows the list of all job runs matching any selected filters.
 */
export default function JobRunsTable(props) {
  const [fetchError, setFetchError] = React.useState('')
  const [isLoaded, setLoaded] = React.useState(false)
  const [rows, setRows] = React.useState([])

  const [filterModel = props.filterModel, setFilterModel] = useQueryParam(
    'filters',
    JsonParam
  )

  const [sortField = props.sortField, setSortField] = useQueryParam(
    'sortField',
    StringParam
  )
  const [sort = props.sort, setSort] = useQueryParam('sort', StringParam)

  const tooltips = {
    S: 'Success',
    F: 'Failure (e2e)',
    f: 'failure (other tests)',
    U: 'upgrade failure',
    I: 'setup failure (installer)',
    N: 'setup failure (infrastructure)',
    n: 'failure before setup (infra)',
    R: 'running',
  }

  const columns = [
    {
      field: 'id',
      hide: true,
      type: 'number',
      filterable: false,
    },
    {
      field: 'timestamp',
      headerName: 'Date / Time',
      filterable: true,
      flex: 1.25,
      type: 'date',
      valueFormatter: (params) => {
        return new Date(params.value)
      },
      renderCell: (params) => {
        return (
          <Tooltip title={relativeTime(new Date(params.value))}>
            <p>{new Date(params.value).toLocaleString()}</p>
          </Tooltip>
        )
      },
    },
    {
      field: 'job',
      headerName: 'Job name',
      flex: props.briefTable ? 1 : 3,
      renderCell: (params) => {
        return (
          <div
            style={{
              display: 'block',
              whiteSpace: 'nowrap',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
            }}
          >
            <Tooltip title={params.value}>
              <Link to={pathForExactJob(props.release, params.value)}>
                {props.briefTable ? params.row.brief_name : params.value}
              </Link>
            </Tooltip>
          </div>
        )
      },
    },
    {
      field: 'testFailures',
      headerName: 'Test Failures',
      type: 'number',
      flex: 0.5,
    },
    {
      field: 'result',
      headerName: 'Result',
      flex: 0.5,
      renderCell: (params) => {
        return (
          <Tooltip title={tooltips[params.value]}>
            <div
              className={'result result-' + params.value}
              style={{ width: '100%', textAlign: 'center' }}
            >
              {params.value}
            </div>
          </Tooltip>
        )
      },
    },
    {
      field: 'url',
      headerName: ' ',
      flex: 0.4,
      renderCell: (params) => {
        return (
          <Tooltip title="View in Prow">
            <Button
              style={{ justifyContent: 'center' }}
              target="_blank"
              startIcon={<DirectionsBoat />}
              href={params.value}
            />
          </Tooltip>
        )
      },
      filterable: false,
    },
    {
      field: 'variants',
      headerName: 'Variants',
      hide: true,
    },
    {
      field: 'failedTestNames',
      headerName: 'Failed tests',
      hide: true,
    },

    // These are fields on the job, not the run - but we can
    // filter by them.
    {
      field: 'name',
      headerName: 'Name',
      type: 'string',
      hide: 'true',
    },
    {
      field: 'tags',
      headerName: 'Tags',
      type: 'array',
      hide: 'true',
    },
    {
      field: 'current_pass_percentage',
      headerName: 'Current pass percentage',
      type: 'number',
      hide: true,
    },
    {
      field: 'current_runs',
      headerName: 'Current runs',
      type: 'number',
      hide: true,
    },
    {
      field: 'previous_runs',
      headerName: 'Previous runs',
      type: 'number',
      hide: true,
    },
    {
      field: 'net_improvement',
      headerName: 'Net improvement',
      type: 'number',
      hide: true,
    },
    {
      field: 'bugs',
      headerName: 'Bug count',
      type: 'number',
      hide: true,
    },
    {
      field: 'associated_bugs',
      headerName: 'Associated bug count',
      type: 'number',
      hide: true,
    },
  ]

  const fetchData = () => {
    let queryString = ''
    if (filterModel && filterModel.items.length > 0) {
      queryString +=
        '&filter=' + encodeURIComponent(JSON.stringify(filterModel))
    }

    if (props.limit > 0) {
      queryString += '&limit=' + encodeURIComponent(props.limit)
    }

    queryString += '&sortField=' + encodeURIComponent(sortField)
    queryString += '&sort=' + encodeURIComponent(sort)

    fetch(
      process.env.REACT_APP_API_URL +
        '/api/jobs/runs?release=' +
        props.release +
        queryString
    )
      .then((response) => {
        if (response.status !== 200) {
          throw new Error('server returned ' + response.status)
        }
        return response.json()
      })
      .then((json) => {
        setRows(json)
        setLoaded(true)
      })
      .catch((error) => {
        setFetchError('Could not retrieve jobs ' + props.release + ', ' + error)
      })
  }

  const requestSearch = (searchValue) => {
    const currentFilters = filterModel
    currentFilters.items = currentFilters.items.filter(
      (f) => f.columnField !== 'job'
    )
    currentFilters.items.push({
      id: 99,
      columnField: 'job',
      operatorValue: 'contains',
      value: searchValue,
    })
    setFilterModel(currentFilters)
  }

  useEffect(() => {
    fetchData()
  }, [filterModel, sort, sortField])

  const pageTitle = () => {
    if (props.title) {
      return (
        <Typography align="center" style={{ margin: 20 }} variant="h4">
          {props.title}
        </Typography>
      )
    }
  }

  if (fetchError !== '') {
    return <Alert severity="error">{fetchError}</Alert>
  }

  if (!isLoaded) {
    return (
      <Backdrop open={!isLoaded}>
        Fetching data...
        <CircularProgress color="inherit" />
      </Backdrop>
    )
  }

  const addFilters = (filter) => {
    const currentFilters = filterModel.items.filter((item) => item.value !== '')

    filter.forEach((item) => {
      if (item.value && item.value !== '') {
        currentFilters.push(item)
      }
    })
    setFilterModel({
      items: currentFilters,
      linkOperator: filterModel.linkOperator || 'and',
    })
  }

  const filterByResult = (result) => {
    const filtersWithoutResult = filterModel.items.filter(
      (i) => i.columnField !== 'result'
    )

    setFilterModel({
      items: [
        ...filtersWithoutResult,
        { columnField: 'result', operatorValue: 'equals', value: result },
      ],
      linkOperator: filterModel.linkOperator || 'and',
    })
  }

  const updateSortModel = (model) => {
    if (model.length === 0) {
      return
    }

    if (sort !== model[0].sort) {
      setSort(model[0].sort)
    }

    if (sortField !== model[0].field) {
      setSortField(model[0].field)
    }
  }

  const legend = (
    <div>
      <span onClick={() => filterByResult('S')} className="legend-item">
        <span className="results results-demo">
          <span className="result result-S">S</span>
        </span>{' '}
        success
      </span>
      <span onClick={() => filterByResult('F')} className="legend-item">
        <span className="results results-demo">
          <span className="result result-F">
            <a onClick={() => filterByResult('F')}>F</a>
          </span>
        </span>{' '}
        failure (e2e)
      </span>
      <span onClick={() => filterByResult('f')} className="legend-item">
        <span className="results results-demo">
          <span className="result result-f">
            <a onClick={() => filterByResult('f')}>f</a>
          </span>
        </span>{' '}
        failure (other tests)
      </span>
      <span onClick={() => filterByResult('U')} className="legend-item">
        <span className="results results-demo">
          <span className="result result-U">U</span>
        </span>{' '}
        upgrade failure
      </span>
      <span onClick={() => filterByResult('I')} className="legend-item">
        <span className="results results-demo">
          <span className="result result-I">I</span>
        </span>{' '}
        setup failure (installer)
      </span>
      <span onClick={() => filterByResult('N')} className="legend-item">
        <span className="results results-demo">
          <span className="result result-N">N</span>
        </span>{' '}
        setup failure (infra)
      </span>
      <span onClick={() => filterByResult('n')} className="legend-item">
        <span className="results results-demo">
          <span className="result result-n">n</span>
        </span>{' '}
        failure before setup (infra)
      </span>
      <span onClick={() => filterByResult('R')} className="legend-item">
        <span className="results results-demo">
          <span className="result result-R">R</span>
        </span>{' '}
        running
      </span>
    </div>
  )

  const table = (
    <DataGrid
      components={{ Toolbar: props.hideControls ? '' : GridToolbar }}
      rows={rows}
      columns={columns}
      autoHeight={true}
      // Filtering:
      filterMode="server"
      sortingOrder={['desc', 'asc']}
      sortModel={[
        {
          field: sortField,
          sort: sort,
        },
      ]}
      // Sorting:
      onSortModelChange={(m) => updateSortModel(m)}
      sortingMode="server"
      pageSize={props.pageSize}
      disableColumnMenu={true}
      rowsPerPageOptions={[5, 10, 25, 50]}
      componentsProps={{
        toolbar: {
          columns: columns,
          clearSearch: () => requestSearch(''),
          doSearch: requestSearch,
          filterModel: filterModel,
          setFilterModel: setFilterModel,
          addFilters: (m) => addFilters(m),
        },
      }}
    />
  )

  if (props.briefTable) {
    return table
  }

  /* eslint-disable react/prop-types */
  return (
    <Fragment>
      {pageTitle()}
      <br />
      <br />
      {legend}
      <Container size="xl" style={{ marginTop: 20 }}>
        {table}
      </Container>
    </Fragment>
  )
}

JobRunsTable.defaultProps = {
  briefTable: false,
  hideControls: false,
  pageSize: 25,
  filterModel: {
    items: [],
  },
  sortField: 'timestamp',
  sort: 'desc',
}

JobRunsTable.propTypes = {
  briefTable: PropTypes.bool,
  classes: PropTypes.object,
  limit: PropTypes.number,
  pageSize: PropTypes.number,
  release: PropTypes.string.isRequired,
  title: PropTypes.string,
  hideControls: PropTypes.bool,
  period: PropTypes.string,
  job: PropTypes.string,
  filterModel: PropTypes.object,
  sort: PropTypes.string,
  sortField: PropTypes.string,
}
