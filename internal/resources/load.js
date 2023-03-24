import http from 'k6/http';
import { sleep } from 'k6';

export const options = {
    scenarios: {
        spike: {
            executor: "ramping-arrival-rate",
            preAllocatedVUs: 3000,
            timeUnit: "0.5s",
            stages: [
                { duration: "10s", target: 10 },
                { duration: "1m", target: 140 },
                { duration: "10s", target: 240 },
                { duration: "5m", target: 240 },
                { duration: "10s", target: 100 },
                { duration: "1m", target: 10 },
                { duration: "10s", target: 0 },
            ],
            gracefulStop: "2m",
        },
    },
};

export default function () {
    const BASE_URL1 = "https://localhost:8050/api/env1";
    const BASE_URL2 = "https://localhost:8050/api/env2";

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
                key: "awesomeFeature",
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
                key: "feature1",
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