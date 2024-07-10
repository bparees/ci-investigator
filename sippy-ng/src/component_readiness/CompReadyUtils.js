import {
  alpha,
  Checkbox,
  FormControl,
  InputBase,
  Typography,
} from '@mui/material'
import { format } from 'date-fns'
import { styled } from '@mui/styles'
import Alert from '@mui/material/Alert'
import green from './green.svg'
import green_half_data from './half.svg'
import green_missing_data from './none.svg'
import heart from './improved.svg'
import React from 'react'
import red from './regressed.svg'
import red_3d from './extreme.svg'
import red_3d_triaged from './extreme-triaged.svg'
import red_triaged from './regressed-triaged.svg'

// Set to true for debug mode
export const debugMode = false

// Make the HH:mm:ss as zeros to be more conducive to caching query caching.
export const dateFormat = 'yyyy-MM-dd 00:00:00'
export const dateEndFormat = 'yyyy-MM-dd 23:59:59'

// This is the table we use when the first page is initially rendered.
export const initialPageTable = {
  rows: [
    {
      component: 'None',
      columns: [
        {
          empty: 'None',
          status: 3, // Let's start with success
          regressed_tests: [],
        },
      ],
    },
  ],
}
export const noDataTable = {
  rows: [
    {
      component: 'No Data found',
      columns: [
        {
          empty: 'None',
          status: 3, // Let's start with success
        },
      ],
    },
  ],
}
export const cancelledDataTable = {
  rows: [
    {
      component: 'Cancelled',
      columns: [
        {
          empty: 'None',
          status: 3, // Let's start with success
        },
      ],
    },
  ],
}
// Make one place to create the Component Readiness api call
export function getAPIUrl() {
  return process.env.REACT_APP_API_URL + '/api/component_readiness'
}

// Make one place to create the Component Readiness test_details api call
export function getTestDetailsAPIUrl() {
  return process.env.REACT_APP_API_URL + '/api/component_readiness/test_details'
}

export const gotoCompReadyMain = () => {
  window.location.href = '/sippy-ng/component_readiness/main'
  //window.history.back()
}

// When we get a fetch error, this will print a standard message.
export function gotFetchError(fetchError) {
  return (
    <Alert severity="error">
      <h2>Failed to load component readiness data</h2>
      <h3>
        {fetchError.split('\n').map((item) => (
          <>
            <hr />
            {item}
          </>
        ))}
      </h3>
      <hr />
      <h3>Check, and possibly fix api server, then click below to retry</h3>
      <button onClick={gotoCompReadyMain}>Retry</button>
    </Alert>
  )
}

// getStatusAndIcon returns a status string and icon to display to denote a visual and textual
// meaning of a 'status' value.  We optionally allow a grayscale mode for the red colors.
export function getStatusAndIcon(status, grayFactor = 0) {
  let icon = ''

  let statusStr = status + ': '

  if (status >= 3) {
    statusStr =
      statusStr + 'SignificantImprovement detected (improved sample rate)'
    icon = (
      <img
        src={heart}
        width="15px"
        height="15px"
        style={{ filter: `grayscale(${grayFactor}%)` }}
      />
    )
  } else if (status == 2) {
    statusStr =
      statusStr + 'Missing Basis And Sample (basis and sample data missing)'
    icon = (
      <img
        src={green_missing_data}
        alt="MissingBasisAndSample"
        width="15px"
        height="15px"
        style={{ filter: `grayscale(${grayFactor}%)` }}
      />
    )
  } else if (status == 1) {
    statusStr = statusStr + 'Missing Basis (basis data missing)'
    icon = (
      <img
        src={green_half_data}
        alt="MissingBasis"
        width="15px"
        height="15px"
        style={{
          filter: `grayscale(${grayFactor}%)`,
        }}
      />
    )
  } else if (status == 0) {
    statusStr = statusStr + 'NoSignificantDifference detected'
    icon = (
      <img
        src={green}
        width="15px"
        height="15px"
        alt="NotSignificant"
        style={{ filter: `grayscale(${grayFactor}%)` }}
      />
    )
  } else if (status == -1) {
    statusStr = statusStr + 'Missing Sample (sample data missing)'
    icon = (
      <img
        src={green_half_data}
        alt="MissingSample"
        width="15px"
        height="15px"
        style={{
          transform: `rotate(180deg)`,
          filter: `grayscale(${grayFactor}%)`,
        }}
      />
    )
  } else if (status == -2) {
    statusStr = statusStr + 'SignificantTriagedRegression detected'
    icon = (
      <img
        width="15px"
        height="15px"
        src={red_triaged}
        alt="SignificantTriagedRegression"
      />
    )
  } else if (status == -3) {
    statusStr =
      statusStr + 'ExtremeTriagedRegression detected ( >15% pass rate change)'
    icon = (
      <img
        width="15px"
        height="15px"
        src={red_3d_triaged}
        alt="ExtremeTriagedRegression >15%"
      />
    )
  } else if (status == -4) {
    statusStr = statusStr + 'SignificantRegression detected'
    icon = (
      <img width="15px" height="15px" src={red} alt="SignificantRegression" />
    )
  } else if (status <= -5) {
    statusStr =
      statusStr + 'ExtremeRegression detected ( >15% pass rate change)'
    icon = (
      <img
        width="15px"
        height="15px"
        src={red_3d}
        alt="ExtremeRegression >15%"
      />
    )
  }

  return [statusStr, icon]
}

// The values of a column's key/value pairs (except status) are
// concatenated to form a column name
export function formColumnName(column) {
  return Object.keys(column)
    .filter(
      (key) =>
        key != 'status' &&
        key != 'regressed_tests' &&
        key != 'triaged_incidents'
    )
    .map((key) => column[key])
    .join(' ')
}

// Given the data pulled from the API server, calculate an array
// of columns using the first row.  Assumption: the number of columns
// is the same across all rows.
// A column looks like this and we concatenate all fields except status:
//   "columns": [
//       {
//         "network": "ovn",
//         "arch": "amd64",
//         "platform": "aws",
//         "status": 0
//       },
// Do our best to handle empty data or a "cancelled" condition.
export function getColumns(data) {
  if (!data || !data.rows || !data.rows[0] || !data.rows[0].component) {
    console.log(
      'data is one of: undefined, no rows, no rows[0], no row[0].component'
    )
    return ['No column']
  }
  if (data.rows[0].component == 'None' || !data.rows[0].columns) {
    return ['No data']
  }
  if (data.rows[0].component === 'Cancelled') {
    console.log('got cancelled')
    return ['Cancelled']
  }

  const firstColumn = data.rows[0].columns
  let columnNames = []
  firstColumn.forEach((column) => {
    const columnValues = formColumnName(column)
    columnNames.push(columnValues)
  })

  return columnNames
}

// The API likes RFC3339 times and the date pickers don't.  So we use this
// function to convert for when we call the API.
// 4 digits, followed by a -, followed by 2 digits, and so on all wrapped in
// a group so we can refer to them as $1 and $2 respectively.
// We add a 'T' in the middle and a 'Z' on the end.
export function makeRFC3339Time(aUrlStr) {
  // Translate all the %20 and %3a into spaces and colons so that the regex can work.
  const decodedStr = decodeURIComponent(aUrlStr)
  // URLSearchParams uses a + to separate date and time.
  const regex = /(\d{4}-\d{2}-\d{2})[\s+](\d{2}:\d{2}:\d{2})/g
  const replaceStr = '$1T$2Z'
  let retVal = decodedStr.replace(regex, replaceStr)

  // The api thinks that the null component is real and will filter accordingly
  // so omit it.
  retVal = retVal.replace(/&component=null/g, '')
  return retVal
}

// Return a formatted date given a long form date from the date picker.
// The given date can be either a long string (from the DatePicker),
// a number (epoch time from when we initialized the start times), or
// a Date object (when called from an event handler function).
export function formatLongDate(aLongDate, aDateFormat) {
  let dateObj
  const typeOfLongDate = typeof aLongDate
  if (typeOfLongDate == 'string' || typeOfLongDate == 'number') {
    dateObj = new Date(aLongDate)
  } else if (typeOfLongDate == 'object') {
    dateObj = aLongDate
  } else {
    // This should never happen, but if it does, try to recover.
    console.log('Error: unknown date format: ', typeof aLongDate)
    dateObj = new Date(aLongDate)
  }
  const ret = format(dateObj, aDateFormat)
  return ret
}

// These next set of variables are used for CompReadyMainInputs

export const groupByList = ['cloud', 'arch', 'network', 'upgrade', 'variants']

// Take the values needed to make an api call and return a string that can be used to
// make that call.
export function getUpdatedUrlParts(
  baseRelease,
  baseStartTime,
  baseEndTime,
  sampleRelease,
  sampleStartTime,
  sampleEndTime,
  groupByCheckedItems,
  excludeCloudsCheckedItems,
  excludeArchesCheckedItems,
  excludeNetworksCheckedItems,
  excludeUpgradesCheckedItems,
  excludeVariantsCheckedItems,
  confidence,
  pity,
  minFail,
  ignoreDisruption,
  ignoreMissing
) {
  const valuesMap = {
    baseRelease: baseRelease,
    baseStartTime: formatLongDate(baseStartTime, dateFormat),
    baseEndTime: formatLongDate(baseEndTime, dateEndFormat),
    sampleRelease: sampleRelease,
    sampleStartTime: formatLongDate(sampleStartTime, dateFormat),
    sampleEndTime: formatLongDate(sampleEndTime, dateEndFormat),
    confidence: confidence,
    pity: pity,
    minFail: minFail,
    ignoreDisruption: ignoreDisruption,
    ignoreMissing: ignoreMissing,
    //component: component,
  }

  const arraysMap = {
    excludeClouds: excludeCloudsCheckedItems,
    excludeArches: excludeArchesCheckedItems,
    excludeNetworks: excludeNetworksCheckedItems,
    excludeUpgrades: excludeUpgradesCheckedItems,
    excludeVariants: excludeVariantsCheckedItems,
    groupBy: groupByCheckedItems,
  }

  const queryParams = new URLSearchParams()

  // Render the plain values first.
  Object.entries(valuesMap).forEach(([key, value]) => {
    queryParams.append(key, value)
  })

  // Render the array values.
  Object.entries(arraysMap).forEach(([key, value]) => {
    if (value && value.length) {
      queryParams.append(key, value.join(','))
    }
  })

  // Stringify and put the begin param character.
  queryParams.sort() // ensure they always stay in sorted order to prevent url history changes

  // When using URLSearchParams to construct a query string, it follows the application/x-www-form-urlencoded format,
  // which uses + to represent space characters. The rest of Sippy uses the URI encoding tools in JS, which relies on
  // %20 for spaces. This makes URL's change, which creates additional history entries, and breaks the back button.
  const queryString = queryParams.toString().replace(/\+/g, '%20')
  const retVal = `?${queryString}`
  return retVal
}

// sortQueryParams sorts a query parameters order so we don't screw up the history when they change
export function sortQueryParams(path) {
  // Split the path into base path and query string
  const [basePath, queryString] = path.split('?')

  if (!queryString) {
    return path
  }

  // Use URLSearchParams to parse and sort the query parameters
  const params = new URLSearchParams(queryString)
  const sortedParams = new URLSearchParams([...params.entries()].sort())

  // Re-assemble the path with sorted query parameters.
  // When using URLSearchParams to construct a query string, it follows the application/x-www-form-urlencoded format,
  // which uses + to represent space characters. The rest of Sippy uses the URI encoding tools in JS, which relies on
  // %20 for spaces. This makes URL's change, which creates additional history entries, and breaks the back button.
  return basePath + '?' + sortedParams.toString().replace(/\+/g, '%20')
}

// Single place to make titles so they look consistent as well as capture the
// key attributes you may want for debugging.
export function makePageTitle(title, ...args) {
  return (
    <Typography variant="h4" style={{ margin: 20, textAlign: 'center' }}>
      <div>{title}</div>
      {debugMode &&
        args.map((item, index) => (
          <div key={index}>
            <Typography variant="body2" component="div" key={index}>
              {item}
            </Typography>
          </div>
        ))}
      <hr />
    </Typography>
  )
}

// Given data and columnNames, calculate an array, parallel to columnNames,
// that has true or false depending on if that column is to be kept and displayed.
// The length and order of the returned array is identical to the columnNames array.
// The criteria for keeping a column is based on the redOnlyChecked checkbox.
// If redOnlyChecked is true, keep columns only if status <= -2
// If redOnlyChecked is false, keep all columns.
export function getKeeperColumns(data, columnNames, redOnlyChecked) {
  let keepColumnList = Array(columnNames.length).fill(
    redOnlyChecked ? false : true
  )

  if (!redOnlyChecked) {
    // All columns are kept and displayed.
    return keepColumnList
  }

  // Do a cross-sectional search across rows/componentsfor status <= -2
  data.rows.forEach((row) => {
    row.columns.forEach((column, index) => {
      // Only if status <= -2 do we keep/display this column.
      if (column.status <= -2) {
        keepColumnList[index] = true
      }
    })
  })
  return keepColumnList
}

export function mergeRegressedTests(data) {
  if (!data || !data.rows || !data.rows[0] || !data.rows[0].component) {
    console.log(
      'data is one of: undefined, no rows, no rows[0], no row[0].component'
    )
    return ['No data']
  }
  if (data.rows[0].component == 'None' || !data.rows[0].columns) {
    return ['No data']
  }
  if (data.rows[0].component === 'Cancelled') {
    console.log('got cancelled')
    return ['Cancelled']
  }

  let regressedTests = []
  data.rows.forEach((row) => {
    row.columns.forEach((column) => {
      if (column.regressed_tests && column.regressed_tests.length > 0) {
        regressedTests = regressedTests.concat(column.regressed_tests)
      }
    })
  })

  regressedTests.sort((a, b) => {
    return (
      a.component.toLowerCase() < b.component.toLowerCase() ||
      a.capability.toLowerCase() < b.capability.toLowerCase()
    )
  })
  regressedTests = regressedTests.map((item, index) => ({ ...item, id: index }))
  return regressedTests
}

export const Search = styled('div')(({ theme }) => ({
  position: 'relative',
  borderRadius: theme.shape.borderRadius,
  backgroundColor: alpha(theme.palette.common.white, 0.15),
  '&:hover': {
    backgroundColor: alpha(theme.palette.common.white, 0.25),
  },
  marginRight: theme.spacing(2),
  marginLeft: 0,
  width: '100%',
  [theme.breakpoints.up('sm')]: {
    marginLeft: theme.spacing(0),
    width: 'auto',
  },
}))

export const SearchIconWrapper = styled('div')(({ theme }) => ({
  padding: theme.spacing(0, 2),
  height: '100%',
  position: 'absolute',
  pointerEvents: 'none',
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
}))

export const StyledInputBase = styled(InputBase)(({ theme }) => ({
  color: 'inherit',
  '& .MuiInputBase-input': {
    padding: theme.spacing(1, 1, 1, 0),
    // vertical padding + font size from searchIcon
    paddingLeft: `calc(1em + ${theme.spacing(4)})`,
    transition: theme.transitions.create('width'),
    width: '100%',
    [theme.breakpoints.up('md')]: {
      width: '20ch',
    },
  },
}))
