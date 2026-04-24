/**
 * Hera WhatsApp Sidecar
 *
 * Connects to WhatsApp via Baileys (multi-device) and communicates
 * with the Hera Go process over a Unix socket using newline-delimited JSON.
 *
 * Protocol (newline-delimited JSON over Unix socket):
 *   Go -> Sidecar: {"type":"send","to":"1234567890@s.whatsapp.net","text":"Hello"}
 *   Sidecar -> Go: {"type":"message","from":"1234567890@s.whatsapp.net","text":"Hi","timestamp":1234567890}
 *   Sidecar -> Go: {"type":"qr","data":"qr-code-string"}
 *   Sidecar -> Go: {"type":"connected","jid":"xxx@s.whatsapp.net"}
 *   Sidecar -> Go: {"type":"disconnected","reason":"..."}
 *
 * Environment variables:
 *   HERA_WA_SOCKET_PATH - Unix socket path (default: /tmp/hera-whatsapp.sock)
 *   HERA_WA_AUTH_DIR    - Auth state directory (default: ~/.config/hera/whatsapp-auth)
 */

const net = require('net');
const fs = require('fs');
const path = require('path');
const os = require('os');

const SOCKET_PATH = process.env.HERA_WA_SOCKET_PATH || '/tmp/hera-whatsapp.sock';
const AUTH_DIR = process.env.HERA_WA_AUTH_DIR || path.join(os.homedir(), '.config', 'hera', 'whatsapp-auth');

let sock = null;
let waSocket = null;
let clients = [];

// Ensure auth directory exists.
fs.mkdirSync(AUTH_DIR, { recursive: true });

/**
 * Send a JSON message to all connected Go clients.
 */
function broadcast(msg) {
    const line = JSON.stringify(msg) + '\n';
    for (const client of clients) {
        try {
            client.write(line);
        } catch (err) {
            console.error('Error writing to client:', err.message);
        }
    }
}

/**
 * Handle a command from the Go process.
 */
async function handleCommand(data) {
    try {
        const cmd = JSON.parse(data);

        switch (cmd.type) {
            case 'send':
                if (!waSocket) {
                    console.error('WhatsApp not connected');
                    return;
                }
                await waSocket.sendMessage(cmd.to, { text: cmd.text });
                console.log(`Sent message to ${cmd.to}`);
                break;

            case 'status':
                broadcast({
                    type: 'status',
                    connected: waSocket !== null,
                    timestamp: Date.now()
                });
                break;

            default:
                console.error('Unknown command type:', cmd.type);
        }
    } catch (err) {
        console.error('Error handling command:', err.message);
    }
}

/**
 * Start the Unix socket server for Go IPC.
 */
function startIPCServer() {
    // Remove stale socket file.
    try { fs.unlinkSync(SOCKET_PATH); } catch (_) { /* ignore */ }

    const server = net.createServer((client) => {
        console.log('Go client connected');
        clients.push(client);

        let buffer = '';

        client.on('data', (chunk) => {
            buffer += chunk.toString();
            const lines = buffer.split('\n');
            buffer = lines.pop(); // Keep incomplete line in buffer.

            for (const line of lines) {
                if (line.trim()) {
                    handleCommand(line.trim());
                }
            }
        });

        client.on('end', () => {
            console.log('Go client disconnected');
            clients = clients.filter(c => c !== client);
        });

        client.on('error', (err) => {
            console.error('Client error:', err.message);
            clients = clients.filter(c => c !== client);
        });
    });

    server.listen(SOCKET_PATH, () => {
        console.log(`IPC server listening on ${SOCKET_PATH}`);
    });

    server.on('error', (err) => {
        console.error('IPC server error:', err.message);
        process.exit(1);
    });

    // Clean up on exit.
    process.on('SIGINT', () => {
        server.close();
        try { fs.unlinkSync(SOCKET_PATH); } catch (_) { /* ignore */ }
        process.exit(0);
    });

    process.on('SIGTERM', () => {
        server.close();
        try { fs.unlinkSync(SOCKET_PATH); } catch (_) { /* ignore */ }
        process.exit(0);
    });
}

/**
 * Initialize and connect to WhatsApp via Baileys.
 */
async function startWhatsApp() {
    let makeWASocket, useMultiFileAuthState, DisconnectReason;
    let qrcode;

    try {
        const baileys = require('@whiskeysockets/baileys');
        makeWASocket = baileys.default || baileys.makeWASocket;
        useMultiFileAuthState = baileys.useMultiFileAuthState;
        DisconnectReason = baileys.DisconnectReason;
    } catch (err) {
        console.error('Baileys not installed. Run: npm install');
        console.error(err.message);
        process.exit(1);
    }

    try {
        qrcode = require('qrcode-terminal');
    } catch (_) {
        qrcode = null;
    }

    const pino = require('pino');
    const logger = pino({ level: 'silent' });

    const { state, saveCreds } = await useMultiFileAuthState(AUTH_DIR);

    async function connectWA() {
        const sock = makeWASocket({
            auth: state,
            printQRInTerminal: false,
            logger: logger,
        });

        sock.ev.on('creds.update', saveCreds);

        sock.ev.on('connection.update', (update) => {
            const { connection, lastDisconnect, qr } = update;

            if (qr) {
                console.log('QR code received. Scan with WhatsApp:');
                if (qrcode) {
                    qrcode.generate(qr, { small: true });
                }
                broadcast({ type: 'qr', data: qr });
            }

            if (connection === 'open') {
                console.log('WhatsApp connected:', sock.user?.id);
                waSocket = sock;
                broadcast({
                    type: 'connected',
                    jid: sock.user?.id || 'unknown'
                });
            }

            if (connection === 'close') {
                waSocket = null;
                const reason = lastDisconnect?.error?.output?.statusCode;
                const shouldReconnect = reason !== DisconnectReason?.loggedOut;

                console.log('WhatsApp disconnected. Reason:', reason);
                broadcast({
                    type: 'disconnected',
                    reason: String(reason || 'unknown')
                });

                if (shouldReconnect) {
                    console.log('Reconnecting in 5 seconds...');
                    setTimeout(connectWA, 5000);
                } else {
                    console.log('Logged out. Delete auth state and restart to re-authenticate.');
                }
            }
        });

        sock.ev.on('messages.upsert', (m) => {
            for (const msg of m.messages) {
                // Skip messages sent by us.
                if (msg.key.fromMe) continue;

                const text = msg.message?.conversation ||
                             msg.message?.extendedTextMessage?.text ||
                             '';

                if (!text) continue;

                const from = msg.key.remoteJid || '';
                const timestamp = msg.messageTimestamp || Math.floor(Date.now() / 1000);

                broadcast({
                    type: 'message',
                    from: from,
                    text: text,
                    timestamp: Number(timestamp),
                    pushName: msg.pushName || ''
                });

                console.log(`Message from ${msg.pushName || from}: ${text.substring(0, 50)}`);
            }
        });

        return sock;
    }

    await connectWA();
}

// Main
console.log('Hera WhatsApp Sidecar starting...');
console.log(`  Socket path: ${SOCKET_PATH}`);
console.log(`  Auth dir: ${AUTH_DIR}`);

startIPCServer();
startWhatsApp().catch((err) => {
    console.error('Failed to start WhatsApp:', err.message);
    console.error('Make sure dependencies are installed: npm install');
});
