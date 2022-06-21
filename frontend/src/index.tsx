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

import React from "react";
import ReactDOM from "react-dom";
import {
    BrowserRouter,
    withRouter,
    RouteComponentProps
} from "react-router-dom";
import { configure, when } from "mobx";

// import "antd/dist/antd.css";
import 'antd/dist/antd.variable.min.css';
import "./index.scss";

import App from "./components/App";
import { appGlobal } from "./state/appGlobal";
import { basePathS, IsBusiness } from "./utils/env";
import { api } from "./state/backendApi";

import './assets/fonts/open-sans.css';
import './assets/fonts/poppins.css';
import './assets/fonts/quicksand.css';
import './assets/fonts/kumbh-sans.css';

import { ConfigProvider } from 'antd';


const exampleColors = {
    antdDefaultBlue: '#1890FF',

    rpBrandSecondaryOrange: '#F15D61', // brand-secondary-color; used in selected menu item
    rpBrandAccentOrange: '#ED6338', // brand-accent-color

    green: '#25B864', // chosen to make it really obvious when changing the theme-color works.
    green2: 'hsl(75deg 100% 50%)',

    debugRed: '#FF0000',
} as const;


ConfigProvider.config({
    theme: {
        primaryColor: "#d70bda",

        infoColor: exampleColors.debugRed,
        successColor: exampleColors.debugRed,
        processingColor: exampleColors.debugRed,
        // errorColor: exampleColors.debugRed,
        warningColor: exampleColors.debugRed,
    },
});


const HistorySetter = withRouter((p: RouteComponentProps) => {
    appGlobal.history = p.history;
    return <></>;
});

// Configure MobX
configure({
    enforceActions: 'never',
    safeDescriptors: true,
});

// Get supported endpoints / kafka cluster version
// In the business version, that endpoint (like any other api endpoint) is
// protected, so we need to delay the call until the user is logged in.
if (!IsBusiness) {
    api.refreshSupportedEndpoints(true);
} else {
    when(() => Boolean(api.userData), () => {
        setImmediate(() => {
            api.refreshSupportedEndpoints(true);
        });
    });
}

ReactDOM.render(
    <BrowserRouter basename={basePathS}>
        <HistorySetter />
        <App />
    </BrowserRouter>,
    document.getElementById("root")
);
