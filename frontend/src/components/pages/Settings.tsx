import React from "react";
import { Section } from "../misc/common";
import { PageComponent, PageInitHelper } from "./Page";

import { motion, AnimatePresence } from "framer-motion"
import { animProps, MotionDiv } from "../../utils/animationProps";
import { observer } from "mobx-react";
import { Checkbox, Alert, Button, Modal, Typography, Select, Row, Col } from "antd";
import { observable } from "mobx";
import { appGlobal } from "../../state/appGlobal";
import { uiSettings } from "../../state/ui";

const { Text } = Typography;

const styleRow = { display: 'table-row', paddingBottom: '0.5em' };
const styleColL = { display: 'table-cell' };
const styleColR = { display: 'table-cell', padding: '0.5em 0', paddingLeft: '1em' };

@observer
export class SettingsPage extends PageComponent {

    @observable test: boolean = true;

    initPage(p: PageInitHelper) {
        p.title = 'Settings';
    }

    render() {
        const user = uiSettings.userDefaults;

        return (<MotionDiv>

            {this.MakeRow('Table pagination controls',
                <Select<typeof user.paginationPosition> value={user.paginationPosition} onChange={v => user.paginationPosition = v} style={{ width: 160 }}>
                    <Select.Option value='top'>Top</Select.Option>
                    <Select.Option value='bottom'>Bottom</Select.Option>
                    <Select.Option value='both'>Both</Select.Option>
                </Select>)}

            {this.MakeRow('2nd setting',
                <Text>hliuhlh</Text>)}

            {/*
            <Alert type='error' message={<>
                <Button type='danger' onClick={this.showDeleteConfirm}>Clear User Settings</Button>
            </>}>
            </Alert>
             */}
        </MotionDiv>
        );
    }

    MakeRow(label: string, content: React.ReactNode) {
        return <>
            <div key={label} style={styleRow}>
                <div style={styleColL}>{label}</div>
                <div style={styleColR}>{content}</div>
            </div>
        </>
    }

    showDeleteConfirm() {
        Modal.confirm({
            title: 'Delete user settings?',
            content: 'This will remove all local Kafka-Owl settings in the browser (clears "localStorage")',
            okText: 'Yes, reset my settings!', okType: 'danger',
            cancelText: "No, don't do anything.",
            width: 700,
            onOk() {
                // todo:
                console.log('clearing settings...');
                appGlobal.history.push(appGlobal.history.location);
            },
            onCancel() {
            },
        });
    }
}
