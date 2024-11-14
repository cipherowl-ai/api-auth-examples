const dotenv = require('dotenv');
const jwt = require('jsonwebtoken');
const fs = require('fs').promises;
const path = require('path');
const os = require('os');
const axios = require('axios');

// Configure environment variables
dotenv.config();

// Configure logging
const logger = {
    debug: (...args) => console.log(new Date().toISOString(), '- DEBUG -', ...args),
    error: (...args) => console.error(new Date().toISOString(), '- ERROR -', ...args)
};

// Constants
const CIPHEROWL_API_URL = 'https://svc.cipherowl.ai';
const CLIENT_ID = process.env.CLIENT_ID;
const CLIENT_SECRET = process.env.CLIENT_SECRET;

var TOKEN_CACHE = {};

async function getTokenFromCache() {
    const token = TOKEN_CACHE.access_token;
    const expiresAt = TOKEN_CACHE.expires_at;
    if (!token || Date.now() / 1000 > expiresAt) {
        return null;
    }

    logger.debug('Get token from cache');
    return token;
}

async function writeTokenToCache(token) {
    const decoded = jwt.decode(token);
    TOKEN_CACHE["access_token"] = token;
    TOKEN_CACHE["expires_at"] = decoded.exp;

    logger.debug('Write token to cache');
}

async function getTokenFromServer() {
    try {
        const url = `${CIPHEROWL_API_URL}/oauth/token`;
        const payload = {
            client_id: CLIENT_ID,
            client_secret: CLIENT_SECRET,
            audience: 'svc.cipherowl.ai',
            grant_type: 'client_credentials'
        };

        const response = await axios.post(url, payload, {
            headers: { 'Content-Type': 'application/json' }
        });

        const token = response.data.access_token;
        logger.debug('Get token from server');
        await writeTokenToCache(token);
        return token;
    } catch (error) {
        throw new Error(`Failed to get token from server: ${error.message}`);
    }
}

async function getToken() {
    // Use token cache to improve performance and reduce server load
    const cachedToken = await getTokenFromCache();
    if (cachedToken) {
        return cachedToken;
    }

    return await getTokenFromServer();
}

async function main() {
    try {
        const project = 'partner';
        const url = `${CIPHEROWL_API_URL}/api/v1/sanction?project=${project}&chain=bitcoin_mainnet&address=12udabs2TkX7NXCSj6KpqXfakjE52ZPLhz`;

        const token = await getToken();
        const response = await axios.get(url, {
            headers: { Authorization: `Bearer ${token}` }
        });

        console.log(JSON.stringify(response.data, null, 4));
    } catch (error) {
        logger.error('Error in main:', error.message);
        process.exit(1);
    }
}

if (require.main === module) {
    main();
}
