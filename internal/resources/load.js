import http from 'k6/http';
import { sleep } from 'k6';

export const options = {
    stages: [
        { duration: '5m', target: 60 }, // simulate ramp-up of traffic from 1 to 60 users over 5 minutes.
        { duration: '10m', target: 60 }, // stay at 60 users for 10 minutes
        { duration: '3m', target: 100 }, // ramp-up to 100 users over 3 minutes (peak hour starts)
        { duration: '2m', target: 100 }, // stay at 100 users for short amount of time (peak hour)
        { duration: '3m', target: 60 }, // ramp-down to 60 users over 3 minutes (peak hour ends)
        { duration: '10m', target: 60 }, // continue at 60 for additional 10 minutes
        { duration: '5m', target: 0 }, // ramp-down to 0 users
    ],
    thresholds: {
        http_req_duration: ['p(99)<1500'], // 99% of requests must complete below 1.5s
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

    sleep(1);
}