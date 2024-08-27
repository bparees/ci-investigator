import { CompReadyVarsProvider } from './component_readiness/CompReadyVars'
import { createTheme, useTheme } from '@mui/material/styles'
import {
  CssBaseline,
  Grid,
  StyledEngineProvider,
  ThemeProvider,
  Tooltip,
  useMediaQuery,
} from '@mui/material'
import { cyan, green, orange, red } from '@mui/material/colors'
import { DarkMode, LightMode } from '@mui/icons-material'
import {
  getReportStartDate,
  getUrlWithoutParams,
  relativeTime,
} from './helpers'
import { JobAnalysis } from './jobs/JobAnalysis'
import { makeStyles, styled } from '@mui/styles'
import { parse, stringify } from 'query-string'
import { QueryParamProvider } from 'use-query-params'
import { ReactRouter5Adapter } from 'use-query-params/adapters/react-router-5'
import { Redirect, Route, Switch } from 'react-router-dom'
import { TestAnalysis } from './tests/TestAnalysis'
import { useCookies } from 'react-cookie'
import Alert from '@mui/material/Alert'
import BuildClusterDetails from './build_clusters/BuildClusterDetails'
import BuildClusterOverview from './build_clusters/BuildClusterOverview'
import ChevronLeftIcon from '@mui/icons-material/ChevronLeft'
import ChevronRightIcon from '@mui/icons-material/ChevronRight'
import ComponentReadiness from './component_readiness/ComponentReadiness'
import Drawer from '@mui/material/Drawer'
import IconButton from '@mui/material/IconButton'
import Install from './releases/Install'
import Jobs from './jobs/Jobs'
import MenuIcon from '@mui/icons-material/Menu'
import MuiAppBar from '@mui/material/AppBar'
import PayloadStream from './releases/PayloadStream'
import PayloadStreams from './releases/PayloadStreams'
import ProwJobRun from './prow_job_runs/ProwJobRun'
import PullRequests from './pull_requests/PullRequests'
import React, { Fragment, useEffect, useMemo } from 'react'
import ReleaseOverview from './releases/ReleaseOverview'
import ReleasePayloadDetails from './releases/ReleasePayloadDetails'
import ReleasePayloads from './releases/ReleasePayloads'
import Repositories from './repositories/Repositories'
import RepositoryDetails from './repositories/RepositoryDetails'
import Sidebar from './components/Sidebar'
import Tests from './tests/Tests'
import Toolbar from '@mui/material/Toolbar'
import Typography from '@mui/material/Typography'
import Upgrades from './releases/Upgrades'
import VariantStatus from './jobs/VariantStatus'

const drawerWidth = 240

const Main = styled('main', { shouldForwardProp: (prop) => prop !== 'open' })(
  ({ theme, open }) => ({
    flexGrow: 1,
    width: `100vw`,
    padding: theme.spacing(3),
    transition: theme.transitions.create('margin', {
      easing: theme.transitions.easing.sharp,
      duration: theme.transitions.duration.leavingScreen,
    }),
    marginLeft: `-${drawerWidth}px`,
    ...(open && {
      width: `calc(100% - ${drawerWidth}px)`,
      transition: theme.transitions.create('margin', {
        easing: theme.transitions.easing.easeOut,
        duration: theme.transitions.duration.enteringScreen,
      }),
      marginLeft: 0,
    }),
  })
)

const AppBar = styled(MuiAppBar, {
  shouldForwardProp: (prop) => prop !== 'open',
})(({ theme, open }) => ({
  transition: theme.transitions.create(['margin', 'width'], {
    easing: theme.transitions.easing.sharp,
    duration: theme.transitions.duration.leavingScreen,
  }),
  ...(open && {
    width: `calc(100% - ${drawerWidth}px)`,
    marginLeft: `${drawerWidth}px`,
    transition: theme.transitions.create(['margin', 'width'], {
      easing: theme.transitions.easing.easeOut,
      duration: theme.transitions.duration.enteringScreen,
    }),
  }),
}))

const DrawerHeader = styled('div')(({ theme }) => ({
  display: 'flex',
  alignItems: 'center',
  padding: theme.spacing(0, 1),
  // necessary for content to be below app bar
  ...theme.mixins.toolbar,
  justifyContent: 'flex-end',
}))

export const ReleasesContext = React.createContext({})
export const CapabilitiesContext = React.createContext([])
export const ReportEndContext = React.createContext('')
const ColorModeContext = React.createContext({ toggleColorMode: () => {} })

// ReleaseGADates is an object that maps each 4.y release to its GA date (if any)
export var ReleaseGADates = {}

const useStyles = makeStyles(() => ({
  root: {
    display: 'flex',
    flexGrow: 1,
  },
  title: {
    flexGrow: 1,
  },
}))

const themes = {
  dark: {
    palette: {
      mode: 'dark',
    },
  },
  light: {
    palette: {
      mode: 'light',
      info: {
        main: cyan[500],
        light: cyan[300],
        dark: cyan[700],
      },
      success: {
        main: green[300],
        light: green[300],
        dark: green[700],
      },
      warning: {
        main: orange[500],
        light: orange[300],
        dark: orange[700],
      },
      error: {
        main: red[500],
        light: red[300],
        dark: red[700],
      },
    },
  },
}

// Default theme, restore settings from v4 color schemes. v5 is much darker.

export default function App(props) {
  const classes = useStyles()
  const theme = useTheme()

  const [cookies, setCookie] = useCookies(['sippyColorMode'])
  const colorModePreference = cookies['sippyColorMode']
  const systemPrefersDark = window.matchMedia(
    '(prefers-color-scheme: dark)'
  ).matches
  const [mode, setMode] = React.useState(
    colorModePreference === 'dark' || colorModePreference === 'light'
      ? colorModePreference
      : systemPrefersDark
      ? 'dark'
      : 'light'
  )

  const colorMode = React.useMemo(
    () => ({
      toggleColorMode: () => {
        setMode((prevMode) => {
          const newMode = prevMode === 'light' ? 'dark' : 'light'
          setCookie('sippyColorMode', newMode, {
            path: '/',
            sameSite: 'Strict',
            expires: new Date('3000-12-31'),
          })
          return newMode
        })
      },
    }),
    [setCookie]
  )

  const [lastUpdated, setLastUpdated] = React.useState(null)
  const [drawerOpen, setDrawerOpen] = React.useState(true)
  const [isLoaded, setLoaded] = React.useState(false)
  const [releases, setReleases] = React.useState({ releases: [] })
  const [capabilities, setCapabilities] = React.useState([])
  const [reportDate, setReportDate] = React.useState([])
  const [fetchError, setFetchError] = React.useState('')

  const fetchData = () => {
    Promise.all([
      fetch(process.env.REACT_APP_API_URL + '/api/releases'),
      fetch(process.env.REACT_APP_API_URL + '/api/capabilities'),
      fetch(process.env.REACT_APP_API_URL + '/api/report_date'),
    ])
      .then(([releases, capabilities, reportDate]) => {
        if (releases.status !== 200) {
          throw new Error('server returned ' + releases.status)
        }

        if (capabilities.status !== 200) {
          throw new Error('server returned ' + capabilities.status)
        }

        if (reportDate.status !== 200) {
          throw new Error('server returned ' + reportDate.status)
        }

        return Promise.all([
          releases.json(),
          capabilities.json(),
          reportDate.json(),
        ])
      })
      .then(([releases, capabilities, reportDate]) => {
        // Remove the Z from the ga_dates so that when Date objects are created,
        // the date is not converted to a local time zone.
        for (const key in releases.ga_dates) {
          if (releases.ga_dates[key]) {
            releases.ga_dates[key] = releases.ga_dates[key].replace('Z', '')
          }
        }
        setReleases(releases)
        setCapabilities(capabilities)
        setReportDate(reportDate['pinnedDateTime'])
        setLastUpdated(new Date(releases.last_updated))
        setLoaded(true)
      })
      .catch((error) => {
        setLoaded(true)
        setFetchError('could not retrieve data:' + error)
      })
  }

  // Disable console.log in production
  useEffect(() => {
    if (process.env.NODE_ENV === 'production') {
      console.log = function () {}
    }
  }, [])

  useEffect(() => {
    if (!isLoaded) {
      fetchData()
    }
  })

  const handleDrawerOpen = () => {
    setDrawerOpen(true)
  }

  const showWithCapability = (capability, el) => {
    if (capabilities.includes(capability)) {
      return el
    }

    return null
  }

  const handleDrawerClose = () => {
    setDrawerOpen(false)
  }

  ReleaseGADates = useMemo(() => {
    let gaDates = {}
    releases.releases
      .filter((aVersion) => {
        // We won't process Presubmits or 3.11
        return aVersion !== 'Presubmits' && aVersion != '3.11'
      })
      .forEach((r) => {
        gaDates[r] = releases.ga_dates[r]
      })
    return gaDates
  }, [releases])

  if (!isLoaded) {
    return <Typography>Loading...</Typography>
  }

  let landingPage = ''
  if (fetchError !== '') {
    landingPage = <Alert severity="error">{fetchError}</Alert>
  } else if (releases?.releases?.length > 0) {
    landingPage = (
      <ReleaseOverview
        key={releases.releases[0]}
        release={releases.releases[0]}
      />
    )
  } else {
    landingPage = 'No releases found! Have you configured Sippy correctly?'
  }

  const startDate = getReportStartDate(reportDate)
  return (
    <ColorModeContext.Provider value={colorMode}>
      <ThemeProvider theme={createTheme(themes[mode])}>
        <StyledEngineProvider injectFirst>
          <ReleasesContext.Provider value={releases}>
            <ReportEndContext.Provider value={reportDate}>
              <CapabilitiesContext.Provider value={capabilities}>
                <CssBaseline />
                <QueryParamProvider
                  ReactRouterRoute={Route}
                  adapter={ReactRouter5Adapter}
                  options={{
                    searchStringToObject: parse,
                    objectToSearchString: stringify,
                  }}
                >
                  <div className={classes.root}>
                    <AppBar
                      position="fixed"
                      open={drawerOpen}
                      sx={{ bgcolor: '#3f51b5' }}
                    >
                      <Toolbar edge="start">
                        <IconButton
                          color="inherit"
                          aria-label="open drawer"
                          onClick={handleDrawerOpen}
                          edge="start"
                          sx={{ mr: 2, ...(drawerOpen && { display: 'none' }) }}
                        >
                          <MenuIcon />
                        </IconButton>
                        <Grid
                          container
                          justifyContent="space-between"
                          alignItems="center"
                        >
                          <Typography variant="h6" className={classes.title}>
                            Sippy
                          </Typography>
                          {showWithCapability(
                            'local_db',
                            <Fragment>
                              Last updated{' '}
                              {lastUpdated !== null
                                ? relativeTime(lastUpdated, startDate)
                                : 'unknown'}
                            </Fragment>
                          )}
                        </Grid>
                      </Toolbar>
                    </AppBar>

                    <Drawer
                      sx={{
                        width: drawerWidth,
                        flexShrink: 0,
                        '& .MuiDrawer-paper': {
                          width: drawerWidth,
                          boxSizing: 'border-box',
                        },
                      }}
                      variant="persistent"
                      anchor="left"
                      open={drawerOpen}
                    >
                      <DrawerHeader>
                        <Tooltip
                          title={
                            mode === 'dark'
                              ? 'Toggle light mode'
                              : 'Toggle dark mode'
                          }
                        >
                          <IconButton
                            sx={{ ml: 1 }}
                            onClick={colorMode.toggleColorMode}
                            color="inherit"
                          >
                            {mode === 'dark' ? <LightMode /> : <DarkMode />}
                          </IconButton>
                        </Tooltip>
                        <IconButton onClick={handleDrawerClose} size="large">
                          {theme.direction === 'ltr' ? (
                            <ChevronLeftIcon />
                          ) : (
                            <ChevronRightIcon />
                          )}
                        </IconButton>
                      </DrawerHeader>
                      <Sidebar releases={releases['releases']} />
                    </Drawer>

                    <Main open={drawerOpen}>
                      <DrawerHeader />
                      {/* eslint-disable react/prop-types */}
                      <Switch>
                        <Route
                          path="/release/:release/tags/:tag"
                          render={(props) => (
                            <ReleasePayloadDetails
                              key={
                                'release-details-' + props.match.params.release
                              }
                              release={props.match.params.release}
                              releaseTag={props.match.params.tag}
                            />
                          )}
                        />

                        <Route
                          path="/release/:release/streams/:arch/:stream"
                          render={(props) => (
                            <PayloadStream
                              release={props.match.params.release}
                              arch={props.match.params.arch}
                              stream={props.match.params.stream}
                            />
                          )}
                        />

                        <Route
                          path="/release/:release/streams"
                          render={(props) => (
                            <PayloadStreams
                              key={
                                'release-streams-' + props.match.params.release
                              }
                              release={props.match.params.release}
                            />
                          )}
                        />

                        <Route
                          path="/release/:release/tags"
                          render={(props) => (
                            <ReleasePayloads
                              key={'release-tags-' + props.match.params.release}
                              release={props.match.params.release}
                            />
                          )}
                        />

                        <Route
                          path="/release/:release"
                          render={(props) => (
                            <ReleaseOverview
                              key={
                                'release-overview-' + props.match.params.release
                              }
                              release={props.match.params.release}
                            />
                          )}
                        />

                        <Route
                          path="/variants/:release/:variant"
                          render={(props) => (
                            <VariantStatus
                              release={props.match.params.release}
                              variant={props.match.params.variant}
                            />
                          )}
                        />

                        <Route
                          path="/jobs/:release/analysis"
                          render={(props) => (
                            <JobAnalysis release={props.match.params.release} />
                          )}
                        />

                        <Route
                          path="/jobs/:release"
                          render={(props) => (
                            <Jobs
                              key={'jobs-' + props.match.params.release}
                              title={
                                'Job results for ' + props.match.params.release
                              }
                              release={props.match.params.release}
                            />
                          )}
                        />

                        <Route
                          path="/tests/:release/analysis"
                          render={(props) => (
                            <TestAnalysis
                              release={props.match.params.release}
                            />
                          )}
                        />

                        <Route
                          path="/tests/:release"
                          render={(props) => (
                            <Tests
                              key={'tests-' + props.match.params.release}
                              release={props.match.params.release}
                            />
                          )}
                        />

                        <Route
                          path="/upgrade/:release"
                          render={(props) => (
                            <Upgrades
                              key={'upgrades-' + props.match.params.release}
                              release={props.match.params.release}
                            />
                          )}
                        />

                        <Route
                          path="/component_readiness"
                          render={(props) => {
                            return (
                              <CompReadyVarsProvider>
                                <ComponentReadiness
                                  key={getUrlWithoutParams(['regressedModal'])}
                                />
                              </CompReadyVarsProvider>
                            )
                          }}
                        />

                        <Route
                          path="/install/:release"
                          render={(props) => (
                            <Install
                              key={'install-' + props.match.params.release}
                              release={props.match.params.release}
                            />
                          )}
                        />

                        <Route
                          path="/build_clusters/:cluster"
                          render={(props) => (
                            <BuildClusterDetails
                              key={'cluster-' + props.match.params.cluster}
                              cluster={props.match.params.cluster}
                            />
                          )}
                        />

                        <Route
                          path="/build_clusters"
                          render={() => <BuildClusterOverview />}
                        />

                        <Route
                          path="/repositories/:release/:org/:repo"
                          render={(props) => (
                            <RepositoryDetails
                              release={props.match.params.release}
                              org={props.match.params.org}
                              repo={props.match.params.repo}
                            />
                          )}
                        />

                        <Route
                          path="/repositories/:release"
                          render={(props) => (
                            <Repositories
                              release={props.match.params.release}
                            />
                          )}
                        />

                        <Route
                          path="/pull_requests/:release"
                          render={(props) => (
                            <PullRequests
                              key={'pr-' + props.match.params.release}
                              release={props.match.params.release}
                            />
                          )}
                        />

                        <Route
                          path="/job_runs/:jobrunid/:jobname?/:repoinfo?/:pullnumber?/intervals"
                          render={(props) => (
                            <ProwJobRun
                              jobRunID={props.match.params.jobrunid}
                              jobName={props.match.params.jobname}
                              repoInfo={props.match.params.repoinfo}
                              pullNumber={props.match.params.pullnumber}
                            />
                          )}
                        />

                        {capabilities.includes('local_db') ? (
                          <Route path="/">{landingPage}</Route>
                        ) : (
                          <Redirect
                            exact
                            from="/"
                            to="/component_readiness/main"
                          />
                        )}
                      </Switch>
                      {/* eslint-enable react/prop-types */}
                    </Main>
                  </div>
                </QueryParamProvider>
              </CapabilitiesContext.Provider>
            </ReportEndContext.Provider>
          </ReleasesContext.Provider>
        </StyledEngineProvider>
      </ThemeProvider>
    </ColorModeContext.Provider>
  )
}
