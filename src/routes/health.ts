import { StatusCodes } from "http-status-codes";
import { Request, Response } from 'express';

export function health(req: Request, res: Response) {
    const data = {
        uptime: process.uptime(),
        message: 'Ok',
        date: new Date()
    }
    res.status(StatusCodes.OK).json(data).end();
}
