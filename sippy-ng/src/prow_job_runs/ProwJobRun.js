import * as lodash from 'lodash'
import {
  ArrayParam,
  BooleanParam,
  encodeQueryParams,
  StringParam,
  useQueryParam,
} from 'use-query-params'
import {
  Button,
  ButtonGroup,
  Checkbox,
  MenuItem,
  Select,
  TextField,
  Tooltip,
} from '@mui/material'
import { CircularProgress } from '@mui/material'
import { stringify } from 'query-string'
import { useHistory } from 'react-router-dom'
import Alert from '@mui/material/Alert'
import FormControlLabel from '@mui/material/FormControlLabel'
import FormGroup from '@mui/material/FormGroup'
import PropTypes from 'prop-types'
import React, { Fragment, useEffect, useState } from 'react'
import TimelineChart from '../components/TimelineChart'

// sourceOrder is our preferred ordering of the sections of the chart (interval sources), assuming that
// source is selected in the UI and present in the intervals file:
const sourceOrder = [
  'OperatorAvailable',
  'OperatorProgressing',
  'OperatorDegraded',
  'NodeState',
  'Disruption',
  'KubeletLog',
  'EtcdLog',
  'EtcdLeadership',
]

const combinedArray = [
  ['AlertInfo', '#fada5e'],
  ['AlertPending', '#fada5e'],
  ['AlertWarning', '#ffa500'],
  ['AlertCritical', '#d0312d'], // alerts
  ['OperatorUnavailable', '#d0312d'],
  ['OperatorDegraded', '#ffa500'],
  ['OperatorProgressing', '#fada5e'], // operators
  ['Update', '#1e7bd9'],
  ['Drain', '#4294e6'],
  ['Reboot', '#6aaef2'],
  ['OperatingSystemUpdate', '#96cbff'],
  ['NodeNotReady', '#fada5e'], // nodes
  ['Passed', '#3cb043'],
  ['Skipped', '#ceba76'],
  ['Flaked', '#ffa500'],
  ['Failed', '#d0312d'], // tests
  ['PodCreated', '#96cbff'],
  ['PodScheduled', '#1e7bd9'],
  ['PodTerminating', '#ffa500'],
  ['ContainerWait', '#ca8dfd'],
  ['ContainerStart', '#9300ff'],
  ['ContainerNotReady', '#fada5e'],
  ['ContainerReady', '#3cb043'],
  ['ContainerReadinessFailed', '#d0312d'],
  ['ContainerReadinessErrored', '#d0312d'],
  ['StartupProbeFailed', '#c90076'], // pods
  ['CIClusterDisruption', '#96cbff'],
  ['Disruption', '#d0312d'], // disruption
  ['Degraded', '#b65049'],
  ['Upgradeable', '#32b8b6'],
  ['False', '#ffffff'],
  ['Unknown', '#bbbbbb'],
  ['PodLogInfo', '#96cbff'],
  ['PodLogWarning', '#fada5e'],
  ['PodLogError', '#d0312d'],
  ['EtcdOther', '#d3d3de'],
  ['EtcdLeaderFound', '#03fc62'],
  ['EtcdLeaderLost', '#fc0303'],
  ['EtcdLeaderElected', '#fada5e'],
  ['EtcdLeaderMissing', '#8c5efa'],
  ['GracefulShutdownInterval', '#6E6E6E'],
  ['CloudMetric', '#6E6E6E'],
]

// While we target a fully dynamic UI that will render whatever origin records if display=true, grouped by Source,
// some Sources/Sections/Groups require specific colors. Entries here are optional.
// The function for each source takes the interval as an argument, and returns a key+color string the chart will then use.
const intervalColorizers = {
  KubeEvent: function (interval) {
    if (interval.message.annotations['pathological'] === 'true') {
      if (interval.message.annotations['interesting'] === 'true') {
        return ['PathologicalKnown', '#0000ff']
      } else {
        return ['PathologicalNew', '#d0312d']
      }
    }
    if (interval.message.annotations['interesting'] === 'true') {
      return ['InterestingEvent', '#6E6E6E']
    }
  },
}

export default function ProwJobRun(props) {
  const history = useHistory()

  const [fetchError, setFetchError] = React.useState('')
  const [isLoaded, setLoaded] = React.useState(false)
  const [eventIntervals, setEventIntervals] = React.useState([])
  const [filteredIntervals, setFilteredIntervals] = React.useState([])

  // Interval colors will hold the colors calculated by invoking intervalColorizers functions
  // against each interval. Anything that matches will get added to this map and passed to the
  // TimelineChart for display. Maps the interval color 'key' to a color string.

  // categories is the set of selected categories to display. It is controlled by a combination
  // of default props, the categories query param, and the buttons the user can modify with.
  const [selectedSources = props.selectedSources, setSelectedSources] =
    useQueryParam('selectedSources', ArrayParam)

  const [
    overrideDisplayFlag = props.overrideDisplayFlag,
    setOverrideDisplayFlag,
  ] = useQueryParam('overrideDisplayFlag', BooleanParam)

  const [allIntervalFiles, setAllIntervalFiles] = useState([])
  const [allSources, setAllSources] = useState([])
  const [intervalFile = props.intervalFile, setIntervalFile] = useState(() => {
    const params = new URLSearchParams(window.location.search)
    if (params.get('intervalFile')) {
      console.log(
        'returning intervalFile from URL search params: ' +
          params.get('intervalFile')
      )
      return params.get('intervalFile')
    }
    return ''
  })

  const [filterText, setFilterText] = useState(() => {
    const params = new URLSearchParams(window.location.search)
    if (params.get('filterText')) {
      return params.get('filterText')
    }
    return ''
  })

  const fetchData = () => {
    let queryString = ''
    console.log(
      'fetching new data: jobRun=' +
        props.jobRunID +
        ', jobName=' +
        props.jobName +
        ', pullNumber=' +
        props.pullNumber +
        ', repoInfo=' +
        props.repoInfo +
        ', intervalFile=' +
        intervalFile
    )

    fetch(
      process.env.REACT_APP_API_URL +
        '/api/jobs/runs/intervals?prow_job_run_id=' +
        props.jobRunID +
        (props.jobName ? '&job_name=' + props.jobName : '') +
        (props.repoInfo ? '&repo_info=' + props.repoInfo : '') +
        (props.pullNumber ? '&pull_number=' + props.pullNumber : '') +
        (intervalFile ? '&file=' + intervalFile : '') +
        queryString
    )
      .then((response) => {
        if (response.status !== 200) {
          throw new Error('server returned ' + response.status)
        }
        return response.json()
      })
      .then((json) => {
        if (json != null) {
          // Process and filter our intervals
          let tmpIntervals = json.items
          mutateIntervals(tmpIntervals)
          setEventIntervals(tmpIntervals)

          let intervalFilesAvailable = json.intervalFilesAvailable
          intervalFilesAvailable.sort()
          setAllIntervalFiles(intervalFilesAvailable)
          let allSources = []
          lodash.forEach(tmpIntervals, function (eventInterval) {
            if (!allSources.includes(eventInterval.source)) {
              allSources.push(eventInterval.source)
            }
          })
          console.log('allSources = ' + allSources)
          setAllSources(allSources)
          // This is a little tricky, we do a query first without specifying a filename, as we don't know what
          // files are available. The server makes a best guess and returns the intervals for that file, as well as
          // a list of all available file names. In the UI if we don't yet have one, populate the select with the value
          // we received.
          if (intervalFile == '') {
            console.log(
              'setting interval file to first intervals filename: ' +
                json.items[0].filename
            )
            // TODO: Causes a duplicate API request when we set this. Look into useRef and only calling
            // fetchData in useEffect if we've made it through an initial page load?
            setIntervalFile(json.items[0].filename)
          }
        } else {
          setEventIntervals([])
        }
        setLoaded(true)
      })
      .catch((error) => {
        setFetchError(
          'Could not retrieve intervals for ' +
            'jobRunID=' +
            props.jobRunID +
            ' jobName=' +
            props.jobName +
            ' pullNumber=' +
            props.pullNumber +
            ' repoInfo=' +
            ', ' +
            error
        )
      })
  }

  useEffect(() => {
    fetchData()
  }, [intervalFile])

  useEffect(() => {
    updateFiltering()
  }, [selectedSources, history, eventIntervals, overrideDisplayFlag])

  useEffect(() => {
    // Delayed processing of the filter text input to allow the user to finish typing before
    // we update our filtering:
    const timer = setTimeout(() => {
      console.log('Filter text updated:', filterText)
      updateFiltering()
    }, 800)

    return () => clearTimeout(timer)
  }, [filterText])

  function updateFiltering() {
    console.log('updating filtering')

    let queryString = encodeQueryParams(
      {
        selectedSources: ArrayParam,
        intervalFile: StringParam,
        filter: StringParam,
        overrideDisplayFlag: BooleanParam,
      },
      { selectedSources, intervalFile, filterText, overrideDisplayFlag }
    )

    history.replace({
      search: stringify(queryString),
    })

    let filteredIntervals = filterIntervals(
      eventIntervals,
      selectedSources,
      filterText,
      overrideDisplayFlag
    )
    setFilteredIntervals(filteredIntervals)
  }

  if (fetchError !== '') {
    return <Alert severity="error">{fetchError}</Alert>
  }

  if (isLoaded === false) {
    return (
      <Fragment>
        <p>
          Loading intervals for job run: jobRunID={props.jobRunID}, jobName=
          {props.jobName}, pullNumber={props.pullNumber}, repoInfo=
          {props.repoInfo}
        </p>
        <CircularProgress />
      </Fragment>
    )
  }

  function groupIntervals(selectedSources, filteredIntervals) {
    let timelineGroups = []

    // Separate sources into those that appear in our explicit ordering and those that don't.
    // Any sources that do not appear in our order list will be added to the end.
    let orderedSources = []
    let otherSources = []

    selectedSources.forEach((source) => {
      if (sourceOrder.includes(source)) {
        orderedSources.push(source)
      } else {
        otherSources.push(source)
      }
    })

    // Sort orderedSources according to sourceOrder
    orderedSources.sort(
      (a, b) => sourceOrder.indexOf(a) - sourceOrder.indexOf(b)
    )

    let finalSourceOrder = orderedSources.concat(otherSources)

    let intervalColors = {}

    finalSourceOrder.forEach((source) => {
      timelineGroups.push({ group: source, data: [] })
      createTimelineData(
        intervalColors,
        source,
        timelineGroups[timelineGroups.length - 1].data,
        filteredIntervals,
        source
      )
    })

    console.log('final intervalColors: ' + JSON.stringify(intervalColors))

    return timelineGroups
  }

  let chartData = groupIntervals(selectedSources, filteredIntervals)

  function handleCategoryClick(buttonValue) {
    console.log('got category button click: ' + buttonValue)
    const newSources = [...selectedSources]
    const selectedIndex = selectedSources.indexOf(buttonValue)

    if (selectedIndex === -1) {
      console.log(buttonValue + ' is now selected')
      newSources.push(buttonValue)
    } else {
      console.log(buttonValue + ' is no longer selected')
      newSources.splice(selectedIndex, 1)
    }

    console.log('new selectedSources: ' + newSources)
    setSelectedSources(newSources)
  }

  const handleIntervalFileChange = (event) => {
    console.log('new interval file selected: ' + event.target.value)
    setIntervalFile(event.target.value)
  }

  const handleFilterChange = (event) => {
    setFilterText(event.target.value)
  }

  // handleSegmentClicked is called whenever an individual interval in the chart is clicked.
  // Used to display details on the interval and locator in a way that a user can copy if needed.
  function handleSegmentClicked(segment) {
    // Copy label to clipboard
    navigator.clipboard.writeText(segment.labelVal)
  }

  const handleOverrideDisplayFlagChanged = (event) => {
    setOverrideDisplayFlag(event.target.checked)
  }

  function segmentTooltipFunc(d) {
    return (
      '<span style="max-inline-size: min-content; display: inline-block;">' +
      '<strong>' +
      d.labelVal +
      '</strong><br/>' +
      '<strong>From: </strong>' +
      new Date(d.timeRange[0]).toUTCString() +
      '<br>' +
      '<strong>To: </strong>' +
      new Date(d.timeRange[1]).toUTCString() +
      '</span>'
    )
  }

  return (
    <Fragment>
      <p>
        Loaded {eventIntervals.length} intervals from GCS, filtered down to{' '}
        {filteredIntervals.length}.
      </p>
      <div>
        Categories:
        <ButtonGroup size="small" aria-label="Categories">
          {allSources.map((source) => (
            <Button
              key={source}
              onClick={() => handleCategoryClick(source)}
              variant={
                selectedSources.includes(source) ? 'contained' : 'outlined'
              }
            >
              {source}
            </Button>
          ))}
        </ButtonGroup>
      </div>
      <div>
        Files:
        <Select
          labelId="interval-file-label"
          id="interval-file"
          value={intervalFile}
          label="Interval File"
          onChange={handleIntervalFileChange}
        >
          {allIntervalFiles.map((iFile) => (
            <MenuItem key={iFile} value={iFile}>
              {iFile}
            </MenuItem>
          ))}
        </Select>
        <TextField
          id="filter"
          label="Regex Filter"
          variant="outlined"
          onChange={handleFilterChange}
          defaultValue={filterText}
        />
      </div>
      <div>
        <Tooltip
          title={
            'Display ALL intervals, not just those that origin indicated were meant for display'
          }
        >
          <FormGroup>
            <FormControlLabel
              checked={overrideDisplayFlag}
              control={<Checkbox onChange={handleOverrideDisplayFlagChanged} />}
              label="Override Display Flag"
            />
          </FormGroup>
        </Tooltip>
      </div>
      <div>
        <TimelineChart
          data={chartData}
          eventIntervals={filteredIntervals}
          segmentClickedFunc={handleSegmentClicked}
          segmentTooltipContentFunc={segmentTooltipFunc}
        />
      </div>
    </Fragment>
  )
}

ProwJobRun.defaultProps = {
  // default list of pre-selected sources:
  selectedSources: [
    'OperatorAvailable',
    'OperatorProgressing',
    'OperatorDegraded',
    'KubeletLog',
    'EtcdLog',
    'EtcdLeadership',
    'Alert',
    'Disruption',
    'E2ETest',
    'APIServerGracefulShutdown',
    'KubeEvent',
    'NodeState',
  ],
  intervalFile: '',
  overrideDisplayFlag: false,
}

ProwJobRun.propTypes = {
  jobRunID: PropTypes.string.isRequired,
  jobName: PropTypes.string,
  repoInfo: PropTypes.string,
  pullNumber: PropTypes.string,
  filterText: PropTypes.string,
  selectedSources: PropTypes.array,
  intervalFile: PropTypes.string,
  overrideDisplayFlag: PropTypes.bool,
}

function filterIntervals(
  eventIntervals,
  selectedSources,
  filterText,
  overrideDisplayFlag
) {
  let re = null
  if (filterText) {
    re = new RegExp(filterText)
  }

  return _.filter(eventIntervals, function (eventInterval) {
    let shouldInclude = false
    if (!selectedSources.includes(eventInterval.source)) {
      return shouldInclude
    }
    if (!overrideDisplayFlag && !eventInterval.display) {
      return shouldInclude
    }
    if (re) {
      if (
        re.test(eventInterval.displayMessage) ||
        re.test(eventInterval.displayLocator)
      ) {
        shouldInclude = true
      }
    } else {
      shouldInclude = true
    }
    return shouldInclude
  })
}

function mutateIntervals(eventIntervals) {
  // Structure the locator data and then categorize the event
  lodash.forEach(eventIntervals, function (eventInterval) {
    // Hack until https://issues.redhat.com/browse/TRT-1653 is fixed, and we don't intend to view old interval files
    // that did not have that fix anymore.
    if (eventInterval.locator.keys === null) {
      eventInterval.locator.keys = {}
    }

    // Hack to split the OperatorSource intervals into "fake" sources of Progressing
    // Available and Degraded:
    if (eventInterval.source === 'OperatorState') {
      if (eventInterval.message.annotations.condition === 'Available') {
        eventInterval.source = 'OperatorAvailable'
      } else if (
        eventInterval.message.annotations.condition === 'Progressing'
      ) {
        eventInterval.source = 'OperatorProgressing'
      } else if (eventInterval.message.annotations.condition === 'Degraded') {
        eventInterval.source = 'OperatorDegraded'
      }
    }

    // Calculate the string representation of the message (tooltip) and locator once on load:
    eventInterval.displayMessage = defaultToolTip(eventInterval)
    eventInterval.displayLocator = buildLocatorDisplayString(
      eventInterval.locator
    )
  })
}

function getDurationString(durationSeconds) {
  const seconds = durationSeconds % 60
  const minutes = Math.floor(durationSeconds / 60)
  let durationString = '['
  if (minutes !== 0) {
    durationString += minutes + 'm'
  }
  durationString += seconds + 's]'
  return durationString
}

function defaultToolTip(item) {
  if (!item.message.annotations) {
    return ''
  }

  const structuredMessage = item.message
  const annotations = structuredMessage.annotations

  const keyValuePairs = Object.entries(annotations).map(([key, value]) => {
    return `${key}/${value}`
  })

  let tt = keyValuePairs.join(' ') + ' ' + structuredMessage.humanMessage

  // TODO: can probably remove this once we're confident all displayed intervals have it set
  if ('display' in item) {
    tt = 'display/' + item.display + ' ' + tt
  }
  if ('source' in item) {
    tt = 'source/' + item.source + ' ' + tt
  }
  tt =
    tt +
    ' ' +
    getDurationString(
      (new Date(item.to).getTime() - new Date(item.from).getTime()) / 1000
    )
  return tt
}

// Used for the actual locators displayed on the right hand side of the chart. Based on the origin go code that does
// similar for whenever we serialize a locator to display.
function buildLocatorDisplayString(i) {
  let keys = Object.keys(i.keys)
  keys = sortKeys(keys)

  let annotations = []
  for (let k of keys) {
    let v = i.keys[k]
    if (k === 'LocatorE2ETestKey') {
      annotations.push(`${k}/${JSON.stringify(v)}`)
    } else {
      annotations.push(`${k}/${v}`)
    }
  }

  return annotations.join(' ')
}

function sortKeys(keys) {
  // Ensure these keys appear in this order. Other keys can be mixed in and will appear at the end in alphabetical order.
  const orderedKeys = [
    'namespace',
    'node',
    'pod',
    'uid',
    'server',
    'container',
    'shutdown',
    'row',
  ]

  // Create a map to store the indices of keys in the orderedKeys array.
  // This will allow us to efficiently check if a key is in orderedKeys and find its position.
  const orderedKeyIndices = {}
  orderedKeys.forEach((key, index) => {
    orderedKeyIndices[key] = index
  })

  // Define a custom sorting function that orders the keys based on the orderedKeys array.
  keys.sort((a, b) => {
    // Get the indices of keys a and b in orderedKeys.
    const indexA = orderedKeyIndices[a]
    const indexB = orderedKeyIndices[b]

    // If both keys exist in orderedKeys, sort them based on their order.
    if (indexA !== undefined && indexB !== undefined) {
      return indexA - indexB
    }

    // If only one of the keys exists in orderedKeys, move it to the front.
    if (indexA !== undefined) {
      return -1
    } else if (indexB !== undefined) {
      return 1
    }

    // If neither key is in orderedKeys, sort alphabetically so we have predictable ordering.
    return a.localeCompare(b)
  })

  return keys
}

function createTimelineData(
  intervalColors,
  timelineVal,
  timelineData,
  filteredEventIntervals,
  source
) {
  const data = {}
  let now = new Date()
  let earliest = filteredEventIntervals.reduce(
    (accumulator, currentValue) =>
      !currentValue.from || accumulator < new Date(currentValue.from)
        ? accumulator
        : new Date(currentValue.from),
    new Date(now.getTime() + 1)
  )
  let latest = filteredEventIntervals.reduce(
    (accumulator, currentValue) =>
      !currentValue.to || accumulator > new Date(currentValue.to)
        ? accumulator
        : new Date(currentValue.to),
    new Date(now.getTime() - 1)
  )
  filteredEventIntervals.forEach((item) => {
    if (item.source !== source) {
      return
    }

    if (intervalColorizers[item.source]) {
      let r = intervalColorizers[item.source](item)
      if (r) {
        console.log('got result: ' + r)
        intervalColors[r[0]] = r[1]
      }
    }

    let startDate = new Date(item.from)
    if (!item.from) {
      startDate = earliest
    }
    let endDate = new Date(item.to)
    if (!item.to) {
      endDate = latest
    }
    let label = item.displayLocator
    let sub = ''
    let val = timelineVal
    if (typeof val === 'function') {
      ;[sub, val] = timelineVal(item)
    }
    let section = data[label]
    if (!section) {
      section = {}
      data[label] = section
    }
    let ranges = section[sub]
    if (!ranges) {
      ranges = []
      section[sub] = ranges
    }
    ranges.push({
      timeRange: [startDate, endDate],
      val: val,
      labelVal: item.displayMessage,
    })
  })
  for (const label in data) {
    const section = data[label]
    for (const sub in section) {
      const data = section[sub]
      const totalDurationSeconds = data.reduce(
        (prev, curr) =>
          prev +
          (curr.timeRange[1].getTime() - curr.timeRange[0].getTime()) / 1000,
        0
      )

      timelineData.push({
        label: label + sub + ' ' + getDurationString(totalDurationSeconds),
        data: data,
      })
    }
  }
}
