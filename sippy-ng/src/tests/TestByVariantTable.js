import './TestByVariantTable.css'
import { grey } from '@mui/material/colors'
import { Link } from 'react-router-dom'
import { makeStyles } from '@mui/styles'
import { parseVariantName, pathForExactTestAnalysis } from '../helpers'
import { scale } from 'chroma-js'
import { TableContainer, Tooltip, Typography } from '@mui/material'
import { useCookies } from 'react-cookie'
import { useTheme } from '@mui/material/styles'
import FormControlLabel from '@mui/material/FormControlLabel'
import FormGroup from '@mui/material/FormGroup'
import HelpOutlineIcon from '@mui/icons-material/HelpOutline'
import Paper from '@mui/material/Paper'
import PassRateIcon from '../components/PassRateIcon'
import PropTypes from 'prop-types'
import React, { Fragment } from 'react'
import Switch from '@mui/material/Switch'
import Table from '@mui/material/Table'
import TableBody from '@mui/material/TableBody'
import TableCell from '@mui/material/TableCell'
import TableHead from '@mui/material/TableHead'
import TableRow from '@mui/material/TableRow'

const useStyles = makeStyles((theme) => ({
  tableContainer: {
    maxHeight: 800,
    [theme.breakpoints.down('sm')]: {
      maxHeight: 500,
    },
    [theme.breakpoints.down('md')]: {
      maxHeight: 600,
    },
    [theme.breakpoints.down('lg')]: {
      maxHeight: 700,
    },
  },
  stickyCellName: {
    position: 'sticky',
    left: 0,
    background: theme.palette.mode === 'dark' ? grey[800] : 'white',
    borderRight: '2px solid black',
    lineHeight: 'normal',
    whiteSpace: 'break-spaces',
    wordWrap: 'break-word',
  },
}))

function PassRateCompare(props) {
  const { previous, current } = props

  return (
    <Fragment>
      {current.toFixed(2)}%
      <PassRateIcon improvement={current - previous} />
      {previous.toFixed(2)}%
    </Fragment>
  )
}

PassRateCompare.propTypes = {
  previous: PropTypes.number,
  current: PropTypes.number,
}

function Cell(props) {
  const { result } = props
  const theme = useTheme()

  const cellBackground = (percent) => {
    const colorScale = scale([
      theme.palette.error.light,
      theme.palette.warning.light,
      theme.palette.success.light,
    ]).domain(props.colorScale)
    return colorScale(percent).hex()
  }

  if (result === undefined) {
    return (
      <Tooltip title="No data">
        <TableCell
          className="cell-result"
          style={{
            textAlign: 'center',
            backgroundColor: theme.palette.text.disabled,
          }}
        >
          <HelpOutlineIcon style={{ color: theme.palette.text.disabled }} />
        </TableCell>
      </Tooltip>
    )
  } else if (result.current_runs === 0) {
    return (
      <Tooltip title="No runs in the current period">
        <TableCell
          className="cell-result"
          style={{
            textAlign: 'center',
            backgroundColor: theme.palette.text.disabled,
          }}
        >
          <HelpOutlineIcon style={{ color: theme.palette.text.disabled }} />
        </TableCell>
      </Tooltip>
    )
  } else if (props.showFull) {
    return (
      <TableCell
        className="cell-result"
        style={{
          textAlign: 'center',
          backgroundColor: cellBackground(result.current_pass_percentage),
        }}
      >
        <PassRateCompare
          current={result.current_pass_percentage}
          previous={result.previous_pass_percentage}
        />
      </TableCell>
    )
  } else {
    return (
      <Tooltip
        title={
          <PassRateCompare
            current={result.current_pass_percentage}
            previous={result.previous_pass_percentage}
          />
        }
      >
        <TableCell
          className="cell-result"
          style={{
            textAlign: 'center',
            backgroundColor: cellBackground(result.current_pass_percentage),
          }}
        >
          <PassRateIcon
            improvement={
              result.current_pass_percentage - result.previous_pass_percentage
            }
          />
        </TableCell>
      </Tooltip>
    )
  }
}

Cell.propTypes = {
  result: PropTypes.object,
  colorScale: PropTypes.array,
  showFull: PropTypes.bool,
  release: PropTypes.string,
  variant: PropTypes.string,
  testName: PropTypes.string,
}

function Row(props) {
  const { columnNames, testName, results } = props
  const classes = useStyles()
  const nameColumn = (
    <TableCell className={classes.stickyCellName} key={testName}>
      <Tooltip title={testName}>
        <Typography className="cell-name">
          <Link
            to={pathForExactTestAnalysis(
              props.release,
              testName,
              props.excludedVariants
            )}
          >
            {testName}
          </Link>
        </Typography>
      </Tooltip>
    </TableCell>
  )

  return (
    <Fragment>
      <TableRow>
        {props.briefTable ? '' : nameColumn}
        {columnNames.map((column, idx) => (
          <Cell
            key={'testName-' + idx}
            colorScale={props.colorScale}
            showFull={props.showFull}
            result={results[column]}
            release={props.release}
            variant={column}
            testName={testName}
          />
        ))}
      </TableRow>
    </Fragment>
  )
}

Row.propTypes = {
  briefTable: PropTypes.bool,
  excludedVariants: PropTypes.array,
  results: PropTypes.object,
  columnNames: PropTypes.array.isRequired,
  testName: PropTypes.string.isRequired,
  colorScale: PropTypes.array.isRequired,
  showFull: PropTypes.bool,
  release: PropTypes.string.isRequired,
}

export default function TestByVariantTable(props) {
  const [cookies, setCookie] = useCookies(['testDetailShowFull'])
  const cookie =
    cookies['testDetailShowFull'] || cookies['testDetailShowFull'] === 'true'
  const [showFull, setShowFull] = React.useState(props.showFull || cookie)
  const classes = useStyles()
  if (props.data === undefined || props.data.tests.length === 0) {
    return <p>No data.</p>
  }

  const handleSwitchFull = (e) => {
    setCookie('testDetailShowFull', e.target.checked, {
      sameSite: 'Strict',
      expires: new Date('3000-12-31'),
    })
    setShowFull(e.target.checked)
  }

  const pageTitle = () => {
    props.title ? (
      <Typography variant="h4" style={{ margin: 20, textAlign: 'center' }}>
        {props.title}
      </Typography>
    ) : null
  }

  if (props.data.tests && Object.keys(props.data.tests).length === 0) {
    return (
      <Fragment>
        {pageTitle}
        <p>No Results.</p>
      </Fragment>
    )
  }

  if (props.data.column_names.length === 0) {
    return (
      <Typography variant="h6" style={{ marginTop: 50 }}>
        No per-variant data found.
      </Typography>
    )
  }

  const nameColumn = (
    <TableCell
      className={`col-name ${props.briefTable ? 'col-hide' : ''}`}
      sx={{ zIndex: 1099 }}
    >
      <FormGroup row>
        <FormControlLabel
          control={
            <Switch
              checked={showFull}
              onChange={handleSwitchFull}
              name="showFull"
            />
          }
          label="Show Full"
        />
      </FormGroup>
    </TableCell>
  )

  return (
    <Paper variant="outlined" elevation={3} sx={{ margin: '20px' }}>
      {pageTitle}
      <TableContainer className={classes.tableContainer}>
        <Table stickyHeader>
          <TableHead>
            <TableRow>
              {props.briefTable ? '' : nameColumn}
              {props.data.column_names.map((column, idx) => {
                const variantInfo = parseVariantName(column)
                return (
                  <TableCell
                    className={'col-result' + (showFull ? '-full' : '')}
                    key={'column' + '-' + idx}
                  >
                    <Typography variant="h6">{variantInfo.name}</Typography>
                    <Typography variant="caption">
                      {variantInfo.variant}
                    </Typography>
                  </TableCell>
                )
              })}
            </TableRow>
          </TableHead>
          <TableBody>
            {Object.keys(props.data.tests).map((test) => (
              <Row
                briefTable={props.briefTable}
                colorScale={props.colorScale}
                showFull={showFull}
                key={test}
                testName={test}
                excludedVariants={props.excludedVariants}
                columnNames={props.data.column_names}
                results={props.data.tests[test]}
                release={props.release}
              />
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </Paper>
  )
}

TestByVariantTable.defaultProps = {
  briefTable: false,
  colorScale: [60, 100],
  excludedVariants: ['never-stable', 'aggregated'],
}

TestByVariantTable.propTypes = {
  briefTable: PropTypes.bool,
  excludedVariants: PropTypes.array,
  columnNames: PropTypes.array,
  current: PropTypes.number,
  data: PropTypes.object,
  previous: PropTypes.number,
  release: PropTypes.string.isRequired,
  results: PropTypes.object,
  testName: PropTypes.string,
  title: PropTypes.string,
  colorScale: PropTypes.array,
  showFull: PropTypes.bool,
}
