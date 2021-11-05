import {
    DataGovernance, createClientWithAutoPoll,
    createClientWithManualPoll, createClientWithLazyLoad
} from 'configcat-node';
import { IConfigCatClient, IConfigCatLogger } from 'configcat-common';
import logger from './Logger'

// Check Configuration
const sdkKey = process.env.CONFIGCAT_SDK_KEY || '';
if (!sdkKey) {
    throw new Error('Invalid CONFIGCAT_SDK_KEY.');
}

class ConfigCatLogger implements IConfigCatLogger {
    log(message: string): void {
        logger.info(message);
    }
    info(message: string): void {
        logger.info(message);
    }
    warn(message: string): void {
        logger.warn(message);
    }
    error(message: string): void {
        logger.err(message);
    }
}

const dataGovernanceEnv = process.env.CONFIGCAT_DATA_GOVERNANCE || 'Global';
let dataGovernance = DataGovernance.Global;
switch (dataGovernanceEnv) {
    case 'Global': dataGovernance = DataGovernance.Global; break;
    case 'EuOnly': dataGovernance = DataGovernance.EuOnly; break;
    default: throw new Error('Invalid CONFIGCAT_DATA_GOVERNANCE value. '
        + 'Possible values: Global, EuOnly.');
}

const requestTimeoutMs = Number(process.env.CONFIGCAT_REQUEST_TIMEOUT_MS || 30000)
if (requestTimeoutMs < 0) {
    throw new Error("Invalid 'CONFIGCAT_REQUEST_TIMEOUT_MS' value.");
}

const pollingMode = process.env.CONFIGCAT_POLLING_MODE || 'AutoPoll';
let configCatClient: IConfigCatClient;
switch (pollingMode) {
    case 'AutoPoll': {
        const pollIntervalSeconds =
            Number(process.env.CONFIGCAT_AUTOPOLL_POLL_INTERVAL_SECONDS || 60);
        // Start the ConfigCatClient
        if (pollIntervalSeconds < 1) {
            throw new Error("Invalid 'CONFIGCAT_AUTOPOLL_POLL_INTERVAL_SECONDS' value.");
        }

        const maxInitWaitTimeSeconds =
            Number(process.env.CONFIGCAT_AUTOPOLL_MAX_INIT_WAIT_TIME_SECONDS || 5);
        // Start the ConfigCatClient
        if (maxInitWaitTimeSeconds < 0) {
            throw new Error("Invalid 'CONFIGCAT_AUTOPOLL_MAX_INIT_WAIT_TIME_SECONDS' value.");
        }

        configCatClient = createClientWithAutoPoll(sdkKey, {
            dataGovernance, requestTimeoutMs, pollIntervalSeconds, maxInitWaitTimeSeconds,
            logger: new ConfigCatLogger()
        });

        break;
    }
    case 'LazyLoad': {
        const cacheTimeToLiveSeconds =
            Number(process.env.CONFIGCAT_LAZYLOAD_CACHE_TIME_TO_LIVE_SECONDS || 60);
        // Start the ConfigCatClient
        if (cacheTimeToLiveSeconds < 1) {
            throw new Error("Invalid 'CONFIGCAT_LAZYLOAD_CACHE_TIME_TO_LIVE_SECONDS' value.");
        }

        configCatClient = createClientWithLazyLoad(sdkKey, {
            dataGovernance, cacheTimeToLiveSeconds, requestTimeoutMs,
            logger: new ConfigCatLogger()
        });
        break;
    }
    case 'ManualPoll': {
        configCatClient = createClientWithManualPoll(sdkKey, {
            dataGovernance, requestTimeoutMs,
            logger: new ConfigCatLogger()
        });
        break;
    }
    default: throw new Error('Invalid CONFIGCAT_POLLING_MODE value. '
        + 'Possible values: AutoPoll, LazyLoad, ManualPoll.');
}

export default configCatClient;