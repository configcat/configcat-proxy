import http from 'k6/http';
import { sleep } from 'k6';

export const options = {
    insecureSkipTLSVerify: true,
    vus: 500,
    duration: '30s',
};

export default () => {
    const responses = http.batch([
        [
            "GET",
            `https://localhost:8050/configuration-files/configcat-proxy/sdk-342/config_v6.json`,
            null,
        ],
        [
            "GET",
            `https://localhost:8050/configuration-files/configcat-proxy/sdk-344/config_v6.json`,
            null,
        ],
    ]);
}