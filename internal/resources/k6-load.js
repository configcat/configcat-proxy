import http from 'k6/http';
import { sleep } from 'k6';

export const options = {
    insecureSkipTLSVerify: true,
    vus: 500,
    duration: '30s',
};

export default () => {
    const BASE_URL1 = "https://localhost:8050/api/sdk-342";
    const BASE_URL2 = "https://localhost:8050/api/sdk-344";

    const payload = JSON.stringify({
        user: {
            Identifier: "09c63c8ad682"
        }
    });

    const responses = http.batch([
        [
            "POST",
            `${BASE_URL1}/eval-all`,
            JSON.stringify({
                user: {
                    Identifier: "09c63c8ad682"
                }
            }),
        ],
        [
            "POST",
            `${BASE_URL1}/eval`,
            JSON.stringify({
                key: "test1",
                user: {
                    Identifier: "09c63c8ad682"
                }
            }),
        ],
        [
            "GET",
            `${BASE_URL1}/keys`,
            null,
        ],
        [
            "POST",
            `${BASE_URL2}/eval-all`,
            JSON.stringify({
                user: {
                    Identifier: "09c63c8ad682"
                }
            }),
        ],
        [
            "POST",
            `${BASE_URL2}/eval`,
            JSON.stringify({
                key: "test2",
                user: {
                    Identifier: "09c63c8ad682"
                }
            }),
        ],
        [
            "GET",
            `${BASE_URL2}/keys`,
            null,
        ],
    ]);
}