import './pre-start'; // Must be the first import
import app from './Server';
import logger from './shared/Logger';
import configCatClient from './shared/ConfigCatClient';

// Register exit events to dispose the configCatClient.
[`exit`, `SIGINT`, `SIGUSR1`, `SIGUSR2`, `uncaughtException`, `SIGTERM`].forEach((eventType) => {
    process.on(eventType, () => {
        if (configCatClient) {
            configCatClient.dispose();
        }
    });
});

// Start the server.
const port = Number(process.env.PORT || 3000);
app.listen(port, () => {
    logger.info('Express server started on port: ' + port);
});
