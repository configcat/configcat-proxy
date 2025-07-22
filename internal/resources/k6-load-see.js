import sse from "k6/x/sse"
import { b64encode } from 'k6/encoding';
import {check} from "k6"

export const options = {
    insecureSkipTLSVerify: true,
    vus: 5000,
    duration: '30s',
};

export default function () {
    const BASE_URL1 = "https://localhost:8050/sse/sdk-342/eval";
    const params = {
        method: 'GET',
    }

    const payload = {
        key: "test1",
        user: {
            Identifier: "09c63c8ad682"
        }
    };

    const data = b64encode(JSON.stringify(payload));
    const url = `${BASE_URL1}/${data}`;

    const response = sse.open(url, params, function (client) {
        client.on('event', function (event) {
            //console.log(`event id=${event.id}, name=${event.name}, data=${event.data}`)
            client.close()
        })

        client.on('error', function (e) {
            console.log('An unexpected error occurred: ', e.error())
        })
    })
    check(response, {"status is 200": (r) => r && r.status === 200})
}