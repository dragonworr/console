import { observable, autorun } from "mobx";
import { touch, assignDeep } from "../utils/utils";
import { DEFAULT_TABLE_PAGE_SIZE } from "../components/misc/common";

const settingsName = 'uiSettings';


/*
	todo:
	- remember UI settings using local storage
	- topic: message filter, display settings, ...
*/

export interface PreviewTag {
    value: string;
    active: boolean;
}

// Settings for an individual topic
export class TopicDetailsSettings {
    topicName: string;

    @observable pageSize = 20;
    @observable activeTabKey: string | undefined = undefined;
    @observable favConfigEntries: string[] = ['cleanup.policy', 'segment.bytes', 'segment.ms'];

    @observable previewTags = [] as PreviewTag[];
    @observable previewTagsCaseSensitive = false;
    @observable previewShowEmptyMessages = true;
    @observable quickSearch = '';
}


const uiSettings = observable({
    sideBarOpen: true,
    selectedClusterIndex: 0,
    perTopicSettings: [] as TopicDetailsSettings[], // don't use directly, instead use uiState.topicDetails

    // todo: refactor into: brokers.list, brokers.detail, topics.messages, topics.config, ...
    brokerList: {
        hideEmptyColumns: false,
        pageSize: DEFAULT_TABLE_PAGE_SIZE,
    },

    topicList: {
        onlyShowChanged: false,
        valueDisplay: 'friendly' as 'friendly' | 'both' | 'raw',
        hideInternalTopics: true,
        pageSize: DEFAULT_TABLE_PAGE_SIZE, // number of topics to show
    },

    consumerGroupList: {
        pageSize: DEFAULT_TABLE_PAGE_SIZE,
        quickSearch: '',
    },

    previewNotificationHideUntil: 0, // utc seconds

    userDefaults: {
        paginationPosition: 'bottom' as ('bottom' | 'top' | 'both'),
    }
});
export { uiSettings };



//
// Settings save/load

// Load settings
let storedSettingsJson = localStorage.getItem(settingsName);
if (storedSettingsJson) {
    const loadedSettings = JSON.parse(storedSettingsJson);
    assignDeep(uiSettings, loadedSettings);
}

// Auto save (timed)
autorun(() => {
    touch(uiSettings);
    const json = JSON.stringify(uiSettings);
    localStorage.setItem(settingsName, json);
    //console.log('settings: ' + json);
}, { delay: 1000 });

// Auto save (on exit)
window.addEventListener('beforeunload', () => {
    const json = JSON.stringify(uiSettings);
    localStorage.setItem(settingsName, json);
    //console.log('settings (unload): ' + json);
});

