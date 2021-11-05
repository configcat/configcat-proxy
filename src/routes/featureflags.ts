import StatusCodes from 'http-status-codes';
import { Request, Response } from 'express';
import configCatClient from "../shared/ConfigCatClient";

/**
 * Get Feature Flag value.
 * req:
 */
export async function getValue(req: Request, res: Response) {
    const { key, defaultValue, user }: {
        key: string, defaultValue: any, user?: {
            identifier: string,
            email?: string,
            country?: string,
            custom?: { [key: string]: string }
        }
    } = req.body;

    if (!key) {
        res.status(StatusCodes.BAD_REQUEST).json({
            error: 'Missing key'
        });
        return;
    }

    if (defaultValue === undefined || defaultValue === null) {
        res.status(StatusCodes.BAD_REQUEST).json({
            error: 'Missing defaultValue'
        });
        return;
    }
    const value = await configCatClient.getValueAsync(key, defaultValue, user);
    return res.status(StatusCodes.OK).json({ value }).end();
}

/**
 * Get all Feature Flag keys.
 */
export async function getAllKeys(req: Request, res: Response) {
    const allkeys = await configCatClient.getAllKeysAsync();
    return res.status(StatusCodes.OK).json(allkeys).end();
}

/**
 * Get all Feature Flag values.
 */
export async function getAllValues(req: Request, res: Response) {
    const { user }: {
        user?: {
            identifier: string,
            email?: string,
            country?: string,
            custom?: { [key: string]: string }
        }
    } = req.body;
    const allValues = await configCatClient.getAllValuesAsync(user);
    return res.status(StatusCodes.OK).json(allValues).end();
}

/**
 * Force refresh the ConfigCatClient cache.
 */
export async function forceRefresh(req: Request, res: Response) {
    await configCatClient.forceRefreshAsync();
    return res.status(StatusCodes.OK).end();
}
