import React, { useState, useEffect } from 'react';

function App() {
  const [agents, setAgents] = useState([]);
  const [command, setCommand] = useState('');
  const [results, setResults] = useState([]);

  const fetchAgents = async () => {
    try {
      const res = await fetch('/api/agents');
      const data = await res.json();
      setAgents(data);
    } catch (err) {
      console.error('Failed to fetch agents', err);
    }
  };

  useEffect(() => {
    fetchAgents();
    const interval = setInterval(fetchAgents, 5000);
    return () => clearInterval(interval);
  }, []);

  const sendCommand = async (agentId) => {
    if (!command.trim()) return;
    try {
      const res = await fetch('/api/tasks', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ agent_id: agentId, command }),
      });
      const data = await res.json();
      setResults((prev) => [...prev, data]);
      setCommand('');
    } catch (err) {
      console.error('Failed to send command', err);
    }
  };

  return (
    <div>
      <h1>Hive Dashboard</h1>
      <div>
        <input
          type="text"
          value={command}
          onChange={(e) => setCommand(e.target.value)}
          placeholder="Enter command..."
        />
      </div>
      <h2>Agents</h2>
      <ul>
        {agents.map((agent) => (
          <li key={agent.id}>
            {agent.id} ({agent.hostname}) — {agent.status}
            <button onClick={() => sendCommand(agent.id)}>Send</button>
          </li>
        ))}
      </ul>
      <h2>Results</h2>
      <ul>
        {results.map((r, i) => (
          <li key={i}>{r.stdout || r.error || 'No output'}</li>
        ))}
      </ul>
    </div>
  );
}

export default App;
