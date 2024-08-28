import './ComponentReadiness.css'
import {
  Box,
  Button,
  Grid,
  Popover,
  TableContainer,
  Tooltip,
  Typography,
} from '@mui/material'
import {
  cancelledDataTable,
  getColumns,
  getStatusAndIcon,
  getSummaryDate,
  getTestDetailsAPIUrl,
  gotFetchError,
  makePageTitle,
  makeRFC3339Time,
  noDataTable,
} from './CompReadyUtils'
import { ComponentReadinessStyleContext } from './ComponentReadiness'
import { CompReadyVarsContext, VarsAtPageLoad } from './CompReadyVars'
import { FileCopy, Help } from '@mui/icons-material'
import { Link } from 'react-router-dom'
import { ReleaseGADates } from '../App'
import { safeEncodeURIComponent } from '../helpers'
import BugButton from '../bugs/BugButton'
import BugTable from '../bugs/BugTable'
import CompReadyCancelled from './CompReadyCancelled'
import CompReadyPageTitle from './CompReadyPageTitle'
import CompReadyProgress from './CompReadyProgress'
import CompReadyTestDetailRow from './CompReadyTestDetailRow'
import CopyPageURL from './CopyPageURL'
import GeneratedAt from './GeneratedAt'
import IconButton from '@mui/material/IconButton'
import InfoIcon from '@mui/icons-material/Info'
import PropTypes from 'prop-types'
import React, { Fragment, useContext, useEffect } from 'react'
import Sidebar from './Sidebar'
import Table from '@mui/material/Table'
import TableBody from '@mui/material/TableBody'
import TableCell from '@mui/material/TableCell'
import TableHead from '@mui/material/TableHead'
import TableRow from '@mui/material/TableRow'

// Big query requests take a while so give the user the option to
// abort in case they inadvertently requested a huge dataset.
let abortController = new AbortController()
const cancelFetch = () => {
  console.log('Aborting page5a')
  abortController.abort()
}

// This component runs when we see /component_readiness/test_details
// This is page 5 which runs when you click a test cell on the right of page 4 or page 4a
export default function CompReadyTestReport(props) {
  const classes = useContext(ComponentReadinessStyleContext)

  const { filterVals, component, capability, testId, environment, testName } =
    props

  const [fetchError, setFetchError] = React.useState('')
  const [isLoaded, setIsLoaded] = React.useState(false)
  const [data, setData] = React.useState({})
  const [showOnlyFailures, setShowOnlyFailures] = React.useState(false)

  // Set the browser tab title
  document.title =
    'Sippy > Component Readiness > Capabilities > Tests > Capability Tests > Test Details' +
    (environment ? `Env` : '')
  const safeComponent = safeEncodeURIComponent(component)
  const safeCapability = safeEncodeURIComponent(capability)
  const safeTestId = safeEncodeURIComponent(testId)

  const appVars = useContext(CompReadyVarsContext)
  const { expandEnvironment } = appVars

  // Helpers for copying the test ID to clipboard
  const [copyPopoverEl, setCopyPopoverEl] = React.useState(null)
  const copyPopoverOpen = Boolean(copyPopoverEl)
  const copyTestID = (event) => {
    event.preventDefault()
    navigator.clipboard.writeText(testId)
    setCopyPopoverEl(event.currentTarget)
    setTimeout(() => setCopyPopoverEl(null), 2000)
  }

  const handleCopy = async (event) => {
    try {
      await navigator.clipboard.writeText(testId)
      setAnchorEl(event.currentTarget)
      setTimeout(() => setAnchorEl(null), 1500) // Close popover after 1.5 seconds
    } catch (err) {
      setAnchorEl(event.currentTarget)
      setTimeout(() => setAnchorEl(null), 1500) // Close popover after 1.5 seconds
    }
  }

  const apiCallStr =
    getTestDetailsAPIUrl() +
    makeRFC3339Time(filterVals) +
    `&component=${safeComponent}` +
    `&capability=${safeCapability}` +
    `&testId=${safeTestId}` +
    (environment ? expandEnvironment(environment) : '')

  useEffect(() => {
    setIsLoaded(false)

    fetch(apiCallStr, { signal: abortController.signal })
      .then((response) => response.json())
      .then((data) => {
        if (data.code < 200 || data.code >= 300) {
          const errorMessage = data.message
            ? `${data.message}`
            : 'No error message'
          throw new Error(`Return code = ${data.code} (${errorMessage})`)
        }
        return data
      })
      .then((json) => {
        // If the basics are not present, consider it no data
        if (!json.component || !json.sample_stats || !json.base_stats) {
          // The api call returned 200 OK but the data was empty
          setData(noDataTable)
        } else {
          setData(json)
        }
      })
      .catch((error) => {
        if (error.name === 'AbortError') {
          setData(cancelledDataTable)

          // Once this fired, we need a new one for the next button click.
          abortController = new AbortController()
        } else {
          setFetchError(`API call failed: ${apiCallStr}\n${error}`)
        }
      })
      .finally(() => {
        setIsLoaded(true)
      })
  }, [])

  if (fetchError !== '') {
    return gotFetchError(fetchError)
  }

  const pageTitle = makePageTitle(
    'Test Details Report',
    environment ? 'page 5a' : 'page 5',
    `component: ${component}`,
    `capability: ${capability}`,
    `testId: ${testId}`,
    `testName: ${testName}`,
    `environment: ${environment}`
  )

  const tableCell = (label, idx) => {
    return (
      <TableCell className={classes.crColResult} key={'column' + '-' + idx}>
        <Typography className={classes.crCellName}>{label}</Typography>
      </TableCell>
    )
  }

  const tableTooltipCell = (label, idx, title) => {
    return (
      <Tooltip title={title}>
        <TableCell className={classes.crColResult} key={'column' + '-' + idx}>
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <InfoIcon style={{ fontSize: '16px', fontWeight: 'lighter' }} />
            <Typography className={classes.crCellName}>{label}</Typography>
          </div>
        </TableCell>
      </Tooltip>
    )
  }

  if (!isLoaded) {
    return <CompReadyProgress apiLink={apiCallStr} cancelFunc={cancelFetch} />
  }

  const columnNames = getColumns(data)
  if (columnNames[0] === 'Cancelled' || columnNames[0] == 'None') {
    return (
      <CompReadyCancelled message={columnNames[0]} apiCallStr={apiCallStr} />
    )
  }

  const handleFailuresOnlyChange = (event) => {
    setShowOnlyFailures(event.target.checked)
  }

  const probabilityStr = (statusStr, fisherNumber) => {
    if (
      statusStr.includes('SignificantRegression') ||
      statusStr.includes('ExtremeRegression')
    ) {
      return `Probability of significant regression: ${(
        (1 - fisherNumber) *
        100
      ).toFixed(2)}%`
    } else if (statusStr.includes('SignificantImprovement')) {
      return `Probability of significant improvement: ${(
        (1 - fisherNumber) *
        100
      ).toFixed(2)}%`
    } else {
      return 'There is no significant evidence of regression'
    }
  }

  const [statusStr, assessmentIcon] = getStatusAndIcon(data.status)
  const significanceTitle = `Test results for individual Prow Jobs may not be statistically
  significant, but when taken in aggregate, there may be a statistically
  significant difference compared to the historical basis
  `

  let url
  if (apiCallStr.startsWith('/')) {
    // In production mode, there is no hostname so we add it so that 'new URL' will work
    // for both production and development modes.
    url = new URL('http://sippy.dptools.openshift.org' + apiCallStr)
  } else {
    url = new URL(apiCallStr)
  }
  const params = new URLSearchParams(url.search)
  const baseStartTime = params.get('baseStartTime')
  const baseEndTime = params.get('baseEndTime')
  const sampleStartTime = params.get('sampleStartTime')
  const sampleEndTime = params.get('sampleEndTime')

  const printParamsAndStats = (
    statsLabel,
    stats,
    from,
    to,
    vCrossCompare,
    variantSelection
  ) => {
    const summaryDate = getSummaryDate(from, to, stats.release, ReleaseGADates)
    return (
      <Fragment>
        {statsLabel} Release: <strong>{stats.release}</strong>
        {summaryDate && (
          <Fragment>
            <br />
            &nbsp;&nbsp;<strong>{summaryDate}</strong>
          </Fragment>
        )}
        <br />
        &nbsp;&nbsp;Start Time: <strong>{from}</strong>
        <br />
        &nbsp;&nbsp;End Time: <strong>{to}</strong>
        <br />
        {vCrossCompare && (
          <Fragment>
            <br />
            &nbsp;&nbsp;Variant Cross Comparison:
            <ul>
              {vCrossCompare.map((group, idx) =>
                variantSelection[group] ? (
                  <li>
                    {group}:&nbsp;
                    <strong>{variantSelection[group].join(', ')}</strong>
                  </li>
                ) : (
                  <li>
                    {group}: <strong>(any)</strong>
                  </li>
                )
              )}
            </ul>
          </Fragment>
        )}
        &nbsp;&nbsp;Statistics:
        <ul>
          <li>Success Rate: {(stats.success_rate * 100).toFixed(2)}%</li>
          <li>Successes: {stats.success_count}</li>
          <li>Failures: {stats.failure_count}</li>
          <li>Flakes: {stats.flake_count}</li>
        </ul>
      </Fragment>
    )
  }

  const printStatsText = (statsLabel, stats, from, to) => {
    return `
${statsLabel} Release: ${stats.release}
Start Time: ${from}
End Time: ${to}
Success Rate: ${(stats.success_rate * 100).toFixed(2)}%
Successes: ${stats.success_count}
Failures: ${stats.failure_count}
Flakes: ${stats.flake_count}`
  }

  return (
    <Fragment>
      <Sidebar isTestDetails={true} />
      <Box
        display="flex"
        justifyContent="right"
        alignItems="right"
        width="100%"
      >
        <Tooltip title="Frequently Asked Questions">
          <Link
            to="/component_readiness/help"
            style={{ textDecoration: 'none' }}
          >
            <IconButton>
              <Help />
            </IconButton>
          </Link>
        </Tooltip>
      </Box>
      <CompReadyPageTitle
        pageTitle={pageTitle}
        pageNumber={5}
        apiCallStr={apiCallStr}
      />
      <h3>
        <Link to="/component_readiness">
          / {environment} &gt; {component}
          &gt; {testName}
        </Link>
      </h3>
      <div align="center" style={{ marginTop: 50 }}>
        <h2>{testName}</h2>
      </div>
      <Grid container>
        <Grid>
          <h2>Linked Bugs</h2>
          <BugTable testName={testName} />
          <Box
            sx={{
              display: 'flex',
              marginTop: 2,
              alignItems: 'center',
              gap: 2,
            }}
          >
            <BugButton
              testName={testName}
              jiraComponentID={data.jira_component_id}
              labels={['component-regression']}
              context={`Component Readiness has found a potential regression in the following test:

{code}${testName}{code}

${probabilityStr(statusStr, data.fisher_exact)}
${printStatsText(
  'Sample (being evaluated)',
  data.sample_stats,
  sampleStartTime,
  sampleEndTime
)}
${printStatsText(
  'Base (historical)',
  data.base_stats,
  baseStartTime,
  baseEndTime
)}

View the test details report at ${document.location.href}
            `}
            />
            <Button
              variant="contained"
              color="secondary"
              href="https://issues.redhat.com/issues/?filter=12432468"
            >
              View other open regressions
            </Button>
          </Box>
        </Grid>
      </Grid>

      <h2>Regression Report</h2>

      <Table>
        <TableBody>
          <TableRow>
            <TableCell>Test ID:</TableCell>
            <TableCell>
              {testId}
              <IconButton
                aria-label="Copy test ID"
                color="inherit"
                onClick={copyTestID}
              >
                <Tooltip title="Copy test ID">
                  <FileCopy />
                </Tooltip>
              </IconButton>
            </TableCell>
          </TableRow>
          <TableRow>
            <TableCell>Environment:</TableCell>
            <TableCell>{environment}</TableCell>
          </TableRow>
          <TableRow>
            <TableCell>Assessment:</TableCell>
            <TableCell>
              <Tooltip title={statusStr}>{assessmentIcon}</Tooltip>
            </TableCell>
          </TableRow>
          <TableRow>
            <TableCell>Probability:</TableCell>
            <TableCell>
              {probabilityStr(statusStr, data.fisher_exact)}
              <Tooltip
                title={`Fisher Exact Number for this basis and sample = ${data.fisher_exact}`}
              >
                <InfoIcon />
              </Tooltip>
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>
      <Grid container spacing={2} style={{ marginTop: '10px' }}>
        <Grid item xs={6}>
          {printParamsAndStats(
            'Basis (historical)',
            data.base_stats,
            VarsAtPageLoad.baseStartTime.toString(),
            VarsAtPageLoad.baseEndTime.toString(),
            VarsAtPageLoad.variantCrossCompare,
            VarsAtPageLoad.includeVariantsCheckedItems
          )}
        </Grid>
        <Grid item xs={6}>
          {printParamsAndStats(
            'Sample (being evaluated)',
            data.sample_stats,
            VarsAtPageLoad.sampleStartTime.toString(),
            VarsAtPageLoad.sampleEndTime.toString(),
            VarsAtPageLoad.variantCrossCompare,
            VarsAtPageLoad.compareVariantsCheckedItems
          )}
        </Grid>
      </Grid>
      <div style={{ marginTop: '10px', marginBottom: '10px' }}>
        <label>
          <input
            type="checkbox"
            checked={showOnlyFailures}
            onChange={handleFailuresOnlyChange}
          />
          Only Show Failures
        </label>
      </div>
      <TableContainer component="div" className="cr-table-wrapper">
        <Table className="cr-comp-read-table">
          <TableHead>
            <TableRow>
              {tableCell('ProwJob Name', 0)}
              {tableCell('Basis Info', 1)}
              {tableCell('Basis Runs', 2)}
              {tableCell('Sample Info', 3)}
              {tableCell('Sample Runs', 4)}
              {tableTooltipCell(
                'Statistically Significant',
                5,
                significanceTitle
              )}
            </TableRow>
          </TableHead>
          <TableBody>
            {/* Ensure we have data before trying to map on it; we need data and rows */}
            {data && data.job_stats && data.job_stats.length > 0 ? (
              data.job_stats
                .sort((a, b) => {
                  if (a.significant && b.significant) {
                    return 0
                  } else if (a.significant) {
                    // This makes it so that statistically significant ones go to the top.
                    return -1
                  } else {
                    return 1
                  }
                })
                .map((element, idx) => {
                  return (
                    <CompReadyTestDetailRow
                      key={idx}
                      element={element}
                      idx={idx}
                      showOnlyFailures={showOnlyFailures}
                    ></CompReadyTestDetailRow>
                  )
                })
            ) : (
              <TableRow>
                {/* No data to render (possible due to a Cancel */}
                <TableCell align="center">No data ; reload to retry</TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </TableContainer>
      <Popover
        id="copyPopover"
        open={copyPopoverOpen}
        anchorEl={copyPopoverEl}
        onClose={() => setCopyPopoverEl(null)}
        anchorOrigin={{
          vertical: 'bottom',
          horizontal: 'center',
        }}
        transformOrigin={{
          vertical: 'top',
          horizontal: 'center',
        }}
      >
        ID copied!
      </Popover>
      <GeneratedAt time={data.generated_at} />
      <CopyPageURL apiCallStr={apiCallStr} />
    </Fragment>
  )
}

CompReadyTestReport.propTypes = {
  filterVals: PropTypes.string.isRequired,
  component: PropTypes.string.isRequired,
  capability: PropTypes.string.isRequired,
  testId: PropTypes.string.isRequired,
  environment: PropTypes.string.isRequired,
  testName: PropTypes.string.isRequired,
}
