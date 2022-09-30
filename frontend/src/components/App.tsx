/**
 * Copyright 2022 Redpanda Data, Inc.
 *
 * Use of this software is governed by the Business Source License
 * included in the file https://github.com/redpanda-data/redpanda/blob/dev/licenses/bsl.md
 *
 * As of the Change Date specified in that file, in accordance with
 * the Business Source License, use of this software will be governed
 * by the Apache License, Version 2.0
 */

import React, { Component, ReactNode } from 'react';
import { observer } from 'mobx-react';
import { Layout, PageHeader, Popover, Button } from 'antd';
import { ColorModeSwitch, Container, Grid, Sidebar } from '@redpanda-data/ui';
import { uiSettings } from '../state/ui';
import { createVisibleSidebarItems, RouteView } from './routes';
import { prettyMilliseconds } from '../utils/utils';
import { api, REST_CACHE_DURATION_SEC } from '../state/backendApi';
import { NavLink, Switch, Route } from 'react-router-dom';
import { Route as AntBreadcrumbRoute } from 'antd/lib/breadcrumb/Breadcrumb';
import { MotionDiv } from '../utils/animationProps';
import { ErrorDisplay } from './misc/ErrorDisplay';
import { uiState } from '../state/uiState';
import { appGlobal } from '../state/appGlobal';
import { ErrorBoundary } from './misc/ErrorBoundary';
import { IsDev, getBasePath, IsCI, AppFeatures } from '../utils/env';
import { UserProfile } from './misc/UserButton';
import fetchWithTimeout from '../utils/fetchWithTimeout';
import { UserData } from '../state/restInterfaces';
import Login from './misc/login';
import LoginCompletePage from './misc/login-complete';
import env, { getBuildDate } from '../utils/env';
import { GithubFilled, TwitterOutlined, LinkedinFilled, SlackSquareOutlined } from '@ant-design/icons';
import { ZeroSizeWrapper, } from '../utils/tsxUtils';
import { UserPreferencesButton } from './misc/UserPreferences';
import { featureErrors } from '../state/supportedFeatures';
import { renderErrorModals } from './misc/ErrorModal';
import { SyncIcon, ChevronRightIcon } from '@primer/octicons-react';
import { isEmbedded } from '../config';
import { ChakraProvider, redpandaTheme } from '@redpanda-data/ui';
import { APP_ROUTES } from './routes';

const { Footer } = Layout;

const VersionInfo = () => {
    const appName = 'Redpanda Console';
    let mode = '';
    if (IsDev) mode = ' - DEV';
    if (IsCI) mode = ' - CI';

    if (env.REACT_APP_CONSOLE_PLATFORM_VERSION)
        mode += ` (Platform Version ${env.REACT_APP_CONSOLE_PLATFORM_VERSION})`;

    let ref = env.REACT_APP_CONSOLE_GIT_REF;
    if (!ref || ref == 'master') ref = '';

    const sha = IsDev
        ? '<no git sha in dev>'
        : env.REACT_APP_CONSOLE_GIT_SHA.slice(0, 7);

    const buildDate = IsDev
        ? new Date()
        : getBuildDate();

    return <>
        <div className="versionTitle">{appName} {mode}</div>
        <div className="versionDate">(built {buildDate?.toLocaleDateString('en-US', { month: 'long', day: '2-digit', year: 'numeric' })})</div>
        <div className="versionGitData">{ref} {sha}</div>
    </>;
};

const SideBar = observer(() => {
    const sidebarItems = createVisibleSidebarItems(APP_ROUTES);
    return (
        <Sidebar items={sidebarItems} isCollapsed={!uiSettings.sideBarOpen}>
            <UserProfile />
        </Sidebar>
    )
});

const AppSide = observer(() => <SideBar  />);


let lastRequestCount = 0;
const DataRefreshButton = observer(() => {

    const spinnerSize = '16px';
    const refreshTextFunc = (): ReactNode => {
        return <div style={{ maxWidth: '350px' }}>
            Click to force a refresh of the data shown in the current page.
            When switching pages, any data older than <span className="codeBox">{prettyMilliseconds(REST_CACHE_DURATION_SEC * 1000)}</span> will be refreshed automatically.
        </div>;
        // TODO: small table that shows what cached data we have and how old it is
    };

    // Track how many requests we've sent in total
    if (api.activeRequests.length == 0) lastRequestCount = 0;
    else lastRequestCount = Math.max(lastRequestCount, api.activeRequests.length);

    const countStr = lastRequestCount > 1
        ? `${lastRequestCount - api.activeRequests.length} / ${lastRequestCount}`
        : '';

    // maybe we need to use the same 'no vertical expansion' trick:
    return <div className="dataRefreshButton">
        {
            api.activeRequests.length == 0
                ?
                <>
                    <Popover title="Force Refresh" content={refreshTextFunc} placement="rightTop" overlayClassName="popoverSmall" >
                        <Button icon={< SyncIcon size={16} />} shape="circle" className="hoverButton" onClick={() => appGlobal.onRefresh()} />
                    </Popover>
                    {/* <span style={{ paddingLeft: '.2em', fontSize: '80%' }}>fetched <b>1 min</b> ago</span> */}
                </>
                :
                <>
                    <span className="spinner" style={{ marginLeft: '8px', width: spinnerSize, height: spinnerSize }} />
                    <span className="pulsating" style={{ padding: '0 10px', fontSize: '80%', userSelect: 'none' }}>Fetching data... {countStr}</span>
                </>
        }
    </div>;
});

const AppPageHeader = observer(() => {

    const breadcrumbs = uiState.pageBreadcrumbs.map(v => ({ path: v.linkTo, breadcrumbName: v.title }));
    const selectedClusterName = uiState.selectedClusterName;
    if (selectedClusterName) {
        //const rootBreadcrumb: AntBreadcrumbRoute = { path: '', breadcrumbName: selectedClusterName };
        const rootBreadcrumb: AntBreadcrumbRoute = { path: '', breadcrumbName: 'Cluster' };
        breadcrumbs.unshift(rootBreadcrumb);
    }

    if (isEmbedded())
        breadcrumbs.splice(0, breadcrumbs.length - 1);

    const breadcrumbRender = (r: AntBreadcrumbRoute, params: any) => (r.breadcrumbName === params.breadcrumbName && r.path === params.path)
        ? <>
            <div className="breadcrumbLast">{r.breadcrumbName}</div>
            <ZeroSizeWrapper justifyContent="start">
                <DataRefreshButton />
            </ZeroSizeWrapper>
        </>
        : <NavLink to={r.path}>{r.breadcrumbName}</NavLink>;

    return <MotionDiv identityKey={uiState.pageTitle} className="pageTitle" style={{ display: 'flex', paddingRight: '16px', alignItems: 'center', marginBottom: '10px' }}>
        <PageHeader
            breadcrumb={{
                routes: breadcrumbs,
                separator: <ZeroSizeWrapper width="10px"><ChevronRightIcon size={14} verticalAlign="unset" /></ZeroSizeWrapper>,
                params: breadcrumbs.last(),
                itemRender: breadcrumbRender
            }}
            title={null}
        />

        <div style={{ marginLeft: 'auto', display: 'flex', alignItems: 'center', gap: '12px' }}>
            <UserPreferencesButton />
            {!isEmbedded() && <ColorModeSwitch/>}
        </div>
    </MotionDiv>;
});

const AppFooter = () => {
    const gitHub = (link: string, title: string) => <>
        <a href={link} title={title} target="_blank" rel="noopener noreferrer">
            <GithubFilled />
        </a>
    </>;

    return <Footer className="footer">
        {/* Social Media Links */}
        <div className="links">
            {isEmbedded() ? gitHub('https://github.com/redpanda-data/redpanda', 'Visit Redpanda\'s GitHub repository') :
                gitHub('https://github.com/redpanda-data/console', 'Visit Redpanda Console\'s GitHub repository')}
            <a href="https://redpanda.com/slack" title="Slack" target="_blank" rel="noopener noreferrer">
                <SlackSquareOutlined />
            </a>
            <a href="https://twitter.com/redpandadata" title="Twitter" target="_blank" rel="noopener noreferrer">
                <TwitterOutlined />
            </a>
            <a href="https://www.linkedin.com/company/redpanda-data" title="LinkedIn" target="_blank" rel="noopener noreferrer">
                <LinkedinFilled />
            </a>
        </div>

        {/* Version Info */}
        <div className="versionText">
            <VersionInfo />
        </div>
    </Footer>;
};

const AppContent = observer(() =>
    <div className="overflowYOverlay" id="mainLayout">

        {/* Page */}
        <LicenseNotification />

        <AppPageHeader />

        <ErrorDisplay>
            <RouteView />
        </ErrorDisplay>

        <AppFooter />

        {/* Currently disabled, read todo comment on UpdatePopup */}
        {/* <UpdatePopup /> */}
        {renderErrorModals()}

    </div>
);

@observer
export default class App extends Component {
    render(): JSX.Element {
        const r = this.loginHandling(); // Complete login, or fetch user if needed
        if (r) return r;

        return (
            <ChakraProvider theme={redpandaTheme}>
                <ErrorBoundary>
                    {/* {IsDev && <DebugDisplay />} */}
                    <Switch>
                        {/* Login (and callbacks) */}
                        <Route exact path="/login" component={Login} />
                        <Route path="/login/callbacks/:provider" render={p => <LoginCompletePage provider={p.match.params.provider} match={p.match} />}></Route>

                        {/* Default View */}
                        <Route path="*">
                            {isEmbedded() ? (
                                <AppContent/>
                            ) : (
                                <Grid templateColumns="auto 1fr" minH="100vh">
                                    <AppSide />
                                    <Container width="full" maxWidth="1500px" as="main" p="8" zIndex={1}>
                                        <AppContent />
                                    </Container>
                                </Grid>
                            )}
                        </Route>
                    </Switch>
                    <FeatureErrorCheck />
                </ErrorBoundary>
            </ChakraProvider>
        );
    }

    loginHandling(): JSX.Element | null {
        if (!AppFeatures.SINGLE_SIGN_ON)
            return null;

        const preLogin = <div style={{ background: 'rgb(233, 233, 233)', height: '100vh' }} />;
        const path = window.location.pathname.removePrefix(getBasePath() ?? '');
        const devPrint = function (str: string) { if (IsDev) console.log(`loginHandling (${path}): ` + str); };

        if (path.startsWith('/login'))
            return null; // already in login process, don't interrupt!

        if (api.userData === null && !path.startsWith('/login')) {
            devPrint('known not logged in, hard redirect');
            window.location.pathname = getBasePath() + '/login'; // definitely not logged in, and in wrong url: hard redirect!
            return preLogin;
        }

        if (api.userData === undefined) {
            devPrint('user is undefined (probably a fresh page load)');

            fetchWithTimeout('./api/users/me', 10 * 1000).then(async r => {
                if (r.ok) {
                    devPrint('user fetched');
                    api.userData = await r.json() as UserData;
                } else if (r.status == 401) { // unauthorized / not logged in
                    devPrint('not logged in');
                    api.userData = null;
                } else if (r.status == 404) { // not found: server must be non-business version
                    devPrint('frontend is configured as business-version, but backend is non-business-version -> will create a local fake user for debugging');
                    uiState.isUsingDebugUserLogin = true;
                    api.userData = {
                        canViewConsoleUsers: false,
                        canListAcls: true,
                        canListQuotas: true,
                        canPatchConfigs: true,
                        canReassignPartitions: true,
                        seat: null as any,
                        user: { providerID: -1, providerName: 'debug provider', id: 'debug', internalIdentifier: 'debug', meta: { avatarUrl: '', email: '', name: 'local fake user for debugging' } }
                    };
                }
            });

            return preLogin;
        } else {
            if (!uiState.isUsingDebugUserLogin)
                devPrint('user is set: ' + JSON.stringify(api.userData));
            return null;
        }
    }
}


@observer
class FeatureErrorCheck extends Component {

    render() {
        if (featureErrors.length > 0) {
            const allErrors = featureErrors.join(' ');
            throw new Error(allErrors);
        }
        return null;
    }
}

@observer
class LicenseNotification extends Component {

    render() {
        if (!api.licenses || !api.licenses.length)
            return null;

        const unixNow = new Date().getTime() / 1000;
        const sourceNames: { [key in string]: string } = {
            'console': 'Console',
            'cluster': 'Cluster',
        };
        const typeNames: { [key in string]: string } = {
            'free_trial': 'Free Trial',
            'open_source': 'Open Source',
            'enterprise': 'Enterprise',
        };

        const withRemainingTime = api.licenses.map(x => {
            const remainingSec = x.expiresAt - unixNow;
            const remainingDays = remainingSec / (60 * 60 * 24);

            const expiredForMoreThanAYear = (remainingSec < 0 && remainingDays < -365);
            const prettyDuration = expiredForMoreThanAYear
                ? 'over a year'
                : prettyMilliseconds(Math.abs(remainingSec) * 1000, { unitCount: 2, verbose: true, secondsDecimalDigits: 0 });

            return {
                ...x,
                remainingSec,
                remainingDays,
                isExpiringSoon: remainingDays < 30,
                isExpired: remainingSec <= 0,
                sourceDisplayName: sourceNames[x.source] ?? x.source,
                typeDisplayName: typeNames[x.type] ?? x.type,
                prettyDuration,
                prettyDateTime: new Date(x.expiresAt * 1000).toLocaleDateString(),
            };
        });

        const warnings = withRemainingTime.filter(x => x.isExpiringSoon || x.isExpired);
        if (!warnings.length)
            return null;

        return <div className="expiringLicenses">
            {warnings.map(e =>
                <div key={e.source}>
                    <div>
                        Your Redpanda Enterprise license (<span className="source">{e.sourceDisplayName}</span>)
                        {e.isExpired
                            ? <> has expired <span className="date">{e.prettyDateTime}</span> ({e.prettyDuration} ago)</>
                            : <> will expire <span className="date">{e.prettyDateTime}</span> ({e.prettyDuration} remaining)</>
                        }
                    </div>
                    <div>
                        To renew your license key, request a new/trial license at:{' '}
                        <a href="https://redpanda.com/license-request" target="_blank" rel="noreferrer">https://redpanda.com/license-request</a>
                    </div>
                </div>
            )}
        </div>
    }
}

