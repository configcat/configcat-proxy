/* eslint-disable @typescript-eslint/no-misused-promises */
import cookieParser from 'cookie-parser';
import helmet from 'helmet';

import express, { NextFunction, Request, Response, Router } from 'express';
import StatusCodes from 'http-status-codes';
import 'express-async-errors';

import logger from './shared/Logger';
import { getValue, getAllValues, getAllKeys, forceRefresh } from './routes/featureflags';
import { health } from './routes/health';

const app = express();
const { BAD_REQUEST } = StatusCodes;


app.use(express.json());
app.use(express.urlencoded({ extended: true }));
app.use(cookieParser());
app.use(helmet());

// Add routes
const router = Router();

router.get('/health', health);
router.post('/getValue', getValue);
router.post('/getAllKeys', getAllKeys);
router.post('/getAllValues', getAllValues);
router.post('/forceRefresh', forceRefresh);

app.use('/', router);


// Print API errors
// eslint-disable-next-line @typescript-eslint/no-unused-vars
app.use((err: Error, req: Request, res: Response, next: NextFunction) => {
    logger.err(err, true);
    return res.status(BAD_REQUEST).json({
        error: err.message,
    });
});


// Export express instance
export default app;

