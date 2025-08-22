import { Client, StatusOK } from 'k6/net/grpc';
import { check, sleep } from 'k6';

const client = new Client();

export const options = {
    vus: 50,
    duration: '30s',
};

export default () => {
    client.connect('127.0.0.1:50051', { reflect: true });
    console.log('connected');

    const data = { sdk_id: 'sdk-342', key: 'test1', user: { Identifier: { string_value: "09c63c8ad682" } } };
    const response = client.invoke('configcat.FlagService/EvalFlag', data);

    check(response, {
        'status is OK': (r) => r && r.status === StatusOK,
    });

    console.log(JSON.stringify(response.message));

    client.close();
    sleep(1);
}