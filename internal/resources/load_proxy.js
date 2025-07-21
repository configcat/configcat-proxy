import http from 'k6/http';
import { sleep } from 'k6';

export const options = {
    insecureSkipTLSVerify: true,
    scenarios: {
        spike: {
            executor: "ramping-arrival-rate",
            preAllocatedVUs: 4000,
            timeUnit: "0.5s",
            stages: [
                { duration: "10s", target: 10 },
                { duration: "1m", target: 240 },
                { duration: "10s", target: 340 },
                { duration: "10s", target: 340 },
                { duration: "10s", target: 640 },
                { duration: "3m", target: 640 },
                { duration: "10s", target: 300 },
                { duration: "10s", target: 300 },
                { duration: "10s", target: 100 },
                { duration: "1m", target: 20 },
                { duration: "10s", target: 0 },
            ],
            gracefulStop: "2m",
        },
    },
};

export default function () {
    const responses = http.batch([
        [
            "GET",
            `https://localhost:8050/configuration-files/sdk1/config_v5.json`,
            null,
        ],
        [
            "GET",
            `https://localhost:8050/configuration-files/sdk2/config_v5.json`,
            null,
        ],
    ]);
}