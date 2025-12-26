// Web Worker script

self.onmessage = (event) => {
    const { type, data } = event.data;

    console.log('[Worker] Received:', type, data);

    switch (type) {
        case 'PING':
            self.postMessage({ type: 'PONG', data: 'PONG! (Response from worker)' });
            break;
        case 'TASK':
            performHeavyTask();
            break;
        default:
            self.postMessage({ type: 'INFO', data: `Unknown message type: ${type}` });
    }
};

function performHeavyTask() {
    self.postMessage({ type: 'INFO', data: 'Worker: Starting heavy computation...' });

    // Simulate a heavy task (e.g., large calculation)
    let result = 0;
    const iterations = 1000000000;
    for (let i = 0; i < iterations; i++) {
        result += Math.sqrt(i);
        // Occasionally report progress
        if (i === iterations / 2) {
            self.postMessage({ type: 'INFO', data: 'Worker: 50% complete...' });
        }
    }

    self.postMessage({ type: 'TASK_COMPLETE', data: `Computation finished! Result: ${result.toFixed(2)}` });
}

// Notify main thread that worker is ready
self.postMessage({ type: 'READY', data: 'Worker initialized and ready.' });
