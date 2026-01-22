import React, { useState, useEffect } from 'react';
import { Activity, Server, Zap, Shield, Database, RefreshCw } from 'lucide-react';
import axios from 'axios';

// Mock data for development when backend is not reachable or CORS issues
const MOCK_STATUS = {
    started: true,
    gateways: {
        "gateway-1": { name: "gateway-1", state: 2, stats: { messages_received: 150, messages_sent: 142, errors: 0 } },
        "gateway-2": { name: "gateway-2", state: 4, stats: { messages_received: 0, messages_sent: 0, errors: 5 } }
    },
    ai: { status: "ready" }
};

function App() {
    const [status, setStatus] = useState(null);
    const [loading, setLoading] = useState(true);
    const [lastUpdated, setLastUpdated] = useState(new Date());

    const fetchStatus = async () => {
        try {
            // In production, this calls the actual API
            // Note: You need to add API Key header if enabled.
            // For now, let's assume we are checking "status" which might be public or we add a key input later.
            // Or we mock it if dev.
            const res = await axios.get('/api/v1/status');
            setStatus(res.data);
        } catch (err) {
            console.warn("Failed to fetch status, using mock", err);
            setStatus(MOCK_STATUS);
        } finally {
            setLoading(false);
            setLastUpdated(new Date());
        }
    };

    useEffect(() => {
        fetchStatus();
        const interval = setInterval(fetchStatus, 3000); // Poll every 3s
        return () => clearInterval(interval);
    }, []);

    const getGatewayStateLabel = (state) => {
        switch (state) {
            case 2: return { text: 'Running', color: 'text-green-400' };
            case 4: return { text: 'Error', color: 'text-red-400' };
            default: return { text: 'Stopped', color: 'text-gray-400' };
        }
    };

    return (
        <div className="min-h-screen bg-transparent p-8">
            {/* Header */}
            <header className="mb-8 flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-cyan-400 to-purple-500">
                        ComX-Bridge
                    </h1>
                    <p className="text-slate-400 text-sm mt-1">Enterprise Protocol Gateway</p>
                </div>
                <div className="flex items-center gap-4">
                    <span className="text-xs text-slate-500">Up: {lastUpdated.toLocaleTimeString()}</span>
                    <div className={`h-3 w-3 rounded-full ${status ? 'bg-green-500 shadow-[0_0_10px_rgba(34,197,94,0.5)]' : 'bg-red-500'}`}></div>
                </div>
            </header>

            <div className="dashboard-grid">
                {/* Stat Cards */}
                <div className="card glass-panel">
                    <div className="flex items-center gap-3 mb-2">
                        <div className="p-2 rounded-lg bg-blue-500/10 text-blue-400">
                            <Server size={20} />
                        </div>
                        <span className="stat-label">Total Gateways</span>
                    </div>
                    <div className="stat-value">
                        {status ? Object.keys(status.gateways).length : '-'}
                    </div>
                </div>

                <div className="card glass-panel">
                    <div className="flex items-center gap-3 mb-2">
                        <div className="p-2 rounded-lg bg-purple-500/10 text-purple-400">
                            <Zap size={20} />
                        </div>
                        <span className="stat-label">AI Engine</span>
                    </div>
                    <div className="stat-value text-xl">
                        {status?.ai?.status === 'ready' ?
                            <span className="text-green-400">Ready</span> :
                            <span className="text-slate-500">Disabled</span>}
                    </div>
                </div>

                <div className="card glass-panel">
                    <div className="flex items-center gap-3 mb-2">
                        <div className="p-2 rounded-lg bg-orange-500/10 text-orange-400">
                            <Database size={20} />
                        </div>
                        <span className="stat-label">Persistence</span>
                    </div>
                    <div className="stat-value text-xl">
                        <span className="text-green-400">Active</span>
                        {/* Ideally fetch from status config */}
                    </div>
                </div>

                <div className="card glass-panel">
                    <div className="flex items-center gap-3 mb-2">
                        <div className="p-2 rounded-lg bg-green-500/10 text-green-400">
                            <Shield size={20} />
                        </div>
                        <span className="stat-label">System Health</span>
                    </div>
                    <div className="stat-value text-xl text-green-400">OK</div>
                </div>
            </div>

            {/* Gateway List */}
            <div className="mt-8">
                <h2 className="text-xl font-semibold mb-4 px-2">Active Gateways</h2>
                <div className="grid grid-cols-1 gap-4">
                    {status && Object.values(status.gateways).map((gw) => {
                        const stateInfo = getGatewayStateLabel(gw.state);
                        return (
                            <div key={gw.name} className="card glass-panel flex items-center justify-between">
                                <div className="flex items-center gap-4">
                                    <div className={`p-3 rounded-xl ${gw.state === 2 ? 'bg-cyan-500/10 text-cyan-400' : 'bg-slate-700/50 text-slate-400'}`}>
                                        <Activity size={24} />
                                    </div>
                                    <div>
                                        <h3 className="text-lg font-medium text-white">{gw.name}</h3>
                                        <div className={`text-sm ${stateInfo.color}`}>{stateInfo.text}</div>
                                    </div>
                                </div>

                                <div className="flex gap-8 text-right">
                                    <div>
                                        <div className="text-xs text-slate-500 uppercase">Received</div>
                                        <div className="font-mono text-lg">{gw.stats?.messages_received || 0}</div>
                                    </div>
                                    <div>
                                        <div className="text-xs text-slate-500 uppercase">Sent</div>
                                        <div className="font-mono text-lg">{gw.stats?.messages_sent || 0}</div>
                                    </div>
                                    <div>
                                        <div className="text-xs text-slate-500 uppercase">Errors</div>
                                        <div className={`font-mono text-lg ${(gw.stats?.errors || 0) > 0 ? 'text-red-400' : 'text-slate-400'}`}>
                                            {gw.stats?.errors || 0}
                                        </div>
                                    </div>
                                </div>
                            </div>
                        );
                    })}
                </div>
            </div>
        </div>
    );
}

export default App;
