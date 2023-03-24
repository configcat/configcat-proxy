import http from 'k6/http';
import { b64encode } from 'k6/encoding';
import { sleep } from 'k6';

export const options = {
    vus: 5000,
    duration: '30s',
};

export default function () {
    const BASE_URL1 = "https://localhost:8050/sse/env1";

    const payload = {
        key: "textsetting",
        user: {
            Identifier: "09c63c8ad682"
        }
    };

    const data = b64encode(JSON.stringify(payload));

    const responses = http.batch([
        [
            "GET",
            `${BASE_URL1}/${data}`,
            null,
        ],
    ]);

    sleep(30);
}