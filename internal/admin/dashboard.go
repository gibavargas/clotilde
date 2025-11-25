package admin

// dashboardHTML is the embedded admin dashboard HTML/JS
const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Clotilde Admin</title>
    <style>
        :root {
            --bg-primary: #0d1117;
            --bg-secondary: #161b22;
            --bg-tertiary: #21262d;
            --border-color: #30363d;
            --text-primary: #c9d1d9;
            --text-secondary: #8b949e;
            --accent-cyan: #58a6ff;
            --accent-green: #3fb950;
            --accent-orange: #d29922;
            --accent-red: #f85149;
            --accent-purple: #a371f7;
            --font-mono: 'JetBrains Mono', 'Fira Code', 'SF Mono', Monaco, monospace;
            --font-sans: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
        }

        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: var(--font-sans);
            background: var(--bg-primary);
            color: var(--text-primary);
            min-height: 100vh;
            line-height: 1.5;
        }

        .container {
            max-width: 1400px;
            margin: 0 auto;
            padding: 24px;
        }

        header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            margin-bottom: 32px;
            padding-bottom: 24px;
            border-bottom: 1px solid var(--border-color);
        }

        .logo {
            display: flex;
            align-items: center;
            gap: 12px;
        }

        .logo-icon {
            width: 40px;
            height: 40px;
            background: linear-gradient(135deg, var(--accent-cyan), var(--accent-purple));
            border-radius: 10px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 20px;
        }

        h1 {
            font-size: 24px;
            font-weight: 600;
            letter-spacing: -0.5px;
        }

        .header-actions {
            display: flex;
            align-items: center;
            gap: 16px;
        }

        .status-badge {
            display: flex;
            align-items: center;
            gap: 6px;
            padding: 6px 12px;
            background: var(--bg-tertiary);
            border-radius: 20px;
            font-size: 13px;
            color: var(--text-secondary);
        }

        .status-dot {
            width: 8px;
            height: 8px;
            background: var(--accent-green);
            border-radius: 50%;
            animation: pulse 2s infinite;
        }

        @keyframes pulse {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.5; }
        }

        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 16px;
            margin-bottom: 32px;
        }

        .stat-card {
            background: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: 12px;
            padding: 20px;
            transition: transform 0.2s, border-color 0.2s;
        }

        .stat-card:hover {
            transform: translateY(-2px);
            border-color: var(--accent-cyan);
        }

        .stat-label {
            font-size: 13px;
            color: var(--text-secondary);
            text-transform: uppercase;
            letter-spacing: 0.5px;
            margin-bottom: 8px;
        }

        .stat-value {
            font-size: 32px;
            font-weight: 700;
            font-family: var(--font-mono);
            background: linear-gradient(135deg, var(--text-primary), var(--accent-cyan));
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
        }

        .stat-subtitle {
            font-size: 12px;
            color: var(--text-secondary);
            margin-top: 4px;
        }

        .section {
            background: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: 12px;
            overflow: hidden;
        }

        .section-header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 16px 20px;
            border-bottom: 1px solid var(--border-color);
            background: var(--bg-tertiary);
        }

        .section-title {
            font-size: 16px;
            font-weight: 600;
            display: flex;
            align-items: center;
            gap: 8px;
        }

        .filters {
            display: flex;
            gap: 12px;
            flex-wrap: wrap;
        }

        select, input[type="date"] {
            background: var(--bg-primary);
            border: 1px solid var(--border-color);
            color: var(--text-primary);
            padding: 8px 12px;
            border-radius: 6px;
            font-size: 13px;
            cursor: pointer;
            transition: border-color 0.2s;
        }

        select:hover, input[type="date"]:hover {
            border-color: var(--accent-cyan);
        }

        select:focus, input[type="date"]:focus {
            outline: none;
            border-color: var(--accent-cyan);
            box-shadow: 0 0 0 3px rgba(88, 166, 255, 0.15);
        }

        .btn {
            background: var(--accent-cyan);
            color: var(--bg-primary);
            border: none;
            padding: 8px 16px;
            border-radius: 6px;
            font-size: 13px;
            font-weight: 500;
            cursor: pointer;
            transition: opacity 0.2s;
        }

        .btn:hover {
            opacity: 0.9;
        }

        .btn-secondary {
            background: var(--bg-tertiary);
            color: var(--text-primary);
            border: 1px solid var(--border-color);
        }

        .btn-small {
            padding: 4px 10px;
            font-size: 11px;
        }

        .logs-table {
            width: 100%;
            border-collapse: collapse;
        }

        .logs-table th {
            text-align: left;
            padding: 12px 16px;
            font-size: 12px;
            font-weight: 600;
            color: var(--text-secondary);
            text-transform: uppercase;
            letter-spacing: 0.5px;
            background: var(--bg-primary);
            border-bottom: 1px solid var(--border-color);
            position: sticky;
            top: 0;
        }

        .logs-table td {
            padding: 12px 16px;
            font-size: 13px;
            border-bottom: 1px solid var(--border-color);
            font-family: var(--font-mono);
            vertical-align: top;
        }

        .logs-table tr:hover {
            background: var(--bg-tertiary);
        }

        .logs-table tr.expanded {
            background: var(--bg-tertiary);
        }

        .logs-table tbody {
            max-height: 500px;
            overflow-y: auto;
        }

        .badge {
            display: inline-block;
            padding: 3px 8px;
            border-radius: 4px;
            font-size: 11px;
            font-weight: 600;
            text-transform: uppercase;
        }

        .badge-success {
            background: rgba(63, 185, 80, 0.15);
            color: var(--accent-green);
        }

        .badge-error {
            background: rgba(248, 81, 73, 0.15);
            color: var(--accent-red);
        }

        .badge-nano {
            background: rgba(163, 113, 247, 0.15);
            color: var(--accent-purple);
        }

        .badge-full {
            background: rgba(210, 153, 34, 0.15);
            color: var(--accent-orange);
        }

        .request-id {
            color: var(--text-secondary);
            font-size: 11px;
        }

        .pagination {
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 16px 20px;
            border-top: 1px solid var(--border-color);
            background: var(--bg-tertiary);
        }

        .pagination-info {
            font-size: 13px;
            color: var(--text-secondary);
        }

        .pagination-buttons {
            display: flex;
            gap: 8px;
        }

        .empty-state {
            text-align: center;
            padding: 60px 20px;
            color: var(--text-secondary);
        }

        .empty-state-icon {
            font-size: 48px;
            margin-bottom: 16px;
            opacity: 0.5;
        }

        .auto-refresh {
            display: flex;
            align-items: center;
            gap: 8px;
            font-size: 13px;
            color: var(--text-secondary);
        }

        .auto-refresh input {
            accent-color: var(--accent-cyan);
        }

        .loading {
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 40px;
        }

        .spinner {
            width: 32px;
            height: 32px;
            border: 3px solid var(--border-color);
            border-top-color: var(--accent-cyan);
            border-radius: 50%;
            animation: spin 1s linear infinite;
        }

        @keyframes spin {
            to { transform: rotate(360deg); }
        }

        /* Detail row styles */
        .detail-row {
            display: none;
        }

        .detail-row.visible {
            display: table-row;
        }

        .detail-cell {
            padding: 0 !important;
            background: var(--bg-primary);
        }

        .detail-content {
            padding: 20px;
            border-top: 1px solid var(--border-color);
        }

        .detail-section {
            margin-bottom: 16px;
        }

        .detail-section:last-child {
            margin-bottom: 0;
        }

        .detail-label {
            font-size: 11px;
            font-weight: 600;
            color: var(--accent-cyan);
            text-transform: uppercase;
            letter-spacing: 0.5px;
            margin-bottom: 8px;
            display: flex;
            align-items: center;
            gap: 6px;
        }

        .detail-text {
            background: var(--bg-tertiary);
            border: 1px solid var(--border-color);
            border-radius: 8px;
            padding: 12px 16px;
            font-family: var(--font-mono);
            font-size: 13px;
            line-height: 1.6;
            white-space: pre-wrap;
            word-break: break-word;
            max-height: 300px;
            overflow-y: auto;
        }

        .detail-text.input {
            border-left: 3px solid var(--accent-purple);
        }

        .detail-text.output {
            border-left: 3px solid var(--accent-green);
        }

        .expand-btn {
            background: transparent;
            border: 1px solid var(--border-color);
            color: var(--text-secondary);
            padding: 4px 8px;
            border-radius: 4px;
            font-size: 11px;
            cursor: pointer;
            transition: all 0.2s;
        }

        .expand-btn:hover {
            border-color: var(--accent-cyan);
            color: var(--accent-cyan);
        }

        .expand-btn.active {
            background: var(--accent-cyan);
            border-color: var(--accent-cyan);
            color: var(--bg-primary);
        }

        @media (max-width: 768px) {
            .container {
                padding: 16px;
            }

            header {
                flex-direction: column;
                align-items: flex-start;
                gap: 16px;
            }

            .filters {
                width: 100%;
            }

            .stat-value {
                font-size: 24px;
            }

            .logs-table {
                display: block;
                overflow-x: auto;
            }

            .grid-row {
                grid-template-columns: 1fr !important;
            }
        }

        /* Settings Styles */
        .settings-card {
            background: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: 12px;
            padding: 24px;
            margin-bottom: 32px;
        }

        .settings-header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            margin-bottom: 24px;
        }

        .settings-title {
            font-size: 18px;
            font-weight: 600;
            display: flex;
            align-items: center;
            gap: 8px;
        }

        .form-group {
            margin-bottom: 20px;
        }

        .form-label {
            display: block;
            font-size: 13px;
            font-weight: 500;
            color: var(--text-secondary);
            margin-bottom: 8px;
        }

        .form-control {
            width: 100%;
            background: var(--bg-primary);
            border: 1px solid var(--border-color);
            color: var(--text-primary);
            padding: 10px 12px;
            border-radius: 6px;
            font-size: 14px;
            font-family: var(--font-sans);
            transition: border-color 0.2s;
        }

        .form-control:focus {
            outline: none;
            border-color: var(--accent-cyan);
            box-shadow: 0 0 0 3px rgba(88, 166, 255, 0.15);
        }

        .textarea-editor {
            font-family: var(--font-mono);
            min-height: 200px;
            resize: vertical;
            line-height: 1.6;
        }

        .save-btn {
            background: var(--accent-green);
            color: #fff;
            border: none;
            padding: 10px 24px;
            border-radius: 6px;
            font-size: 14px;
            font-weight: 600;
            cursor: pointer;
            display: flex;
            align-items: center;
            gap: 8px;
            transition: opacity 0.2s;
        }

        .save-btn:hover {
            opacity: 0.9;
        }

        .save-btn:disabled {
            opacity: 0.5;
            cursor: not-allowed;
        }

        .toast {
            position: fixed;
            bottom: 24px;
            right: 24px;
            padding: 12px 24px;
            border-radius: 8px;
            background: var(--bg-tertiary);
            border: 1px solid var(--border-color);
            color: var(--text-primary);
            font-size: 14px;
            transform: translateY(100px);
            opacity: 0;
            transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
            z-index: 1000;
            display: flex;
            align-items: center;
            gap: 12px;
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
        }

        .toast.show {
            transform: translateY(0);
            opacity: 1;
        }

        .toast-success {
            border-left: 4px solid var(--accent-green);
        }

        .toast-error {
            border-left: 4px solid var(--accent-red);
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <div class="logo">
                <div class="logo-icon">🚗</div>
                <h1>Clotilde Admin</h1>
            </div>
            <div class="header-actions">
                <div class="auto-refresh">
                    <input type="checkbox" id="autoRefresh" checked>
                    <label for="autoRefresh">Auto-refresh (10s)</label>
                </div>
                <div class="status-badge">
                    <span class="status-dot"></span>
                    <span id="uptime">Loading...</span>
                </div>
            </div>
        </header>

        <div class="stats-grid" id="statsGrid">
            <div class="stat-card">
                <div class="stat-label">Requests Today</div>
                <div class="stat-value" id="requestsToday">-</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Total Requests</div>
                <div class="stat-value" id="totalRequests">-</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Avg Response Time</div>
                <div class="stat-value" id="avgResponseTime">-</div>
                <div class="stat-subtitle">milliseconds</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Error Rate</div>
                <div class="stat-value" id="errorRate">-</div>
                <div class="stat-subtitle">percentage</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Model: Nano</div>
                <div class="stat-value" id="modelNano">-</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Model: Full</div>
                <div class="stat-value" id="modelFull">-</div>
            </div>
        </div>

        <div class="settings-card">
            <div class="settings-header">
                <div class="settings-title">
                    ⚙️ Configuration
                </div>
                <button class="save-btn" id="saveConfigBtn" onclick="saveConfig()">
                    <span class="btn-text">Save Changes</span>
                    <div class="spinner" style="width: 16px; height: 16px; border-width: 2px; display: none;"></div>
                </button>
            </div>

            <div class="grid-row" style="display: grid; grid-template-columns: 1fr 1fr; gap: 24px; margin-bottom: 24px;">
                <div class="form-group">
                    <label class="form-label">Standard Model (Simple/Fast)</label>
                    <select class="form-control" id="standardModel">
                        <optgroup label="Confirmed Working">
                            <option value="gpt-4o-mini">gpt-4o-mini (Recommended)</option>
                            <option value="gpt-4o">gpt-4o</option>
                            <option value="gpt-4-turbo">gpt-4-turbo</option>
                            <option value="gpt-3.5-turbo">gpt-3.5-turbo</option>
                        </optgroup>
                        <optgroup label="GPT-4.1 Series">
                            <option value="gpt-4.1-nano">gpt-4.1-nano</option>
                            <option value="gpt-4.1-mini">gpt-4.1-mini</option>
                            <option value="gpt-4.1">gpt-4.1</option>
                        </optgroup>
                        <optgroup label="GPT-5 Series">
                            <option value="gpt-5-nano">gpt-5-nano</option>
                            <option value="gpt-5-mini">gpt-5-mini</option>
                            <option value="gpt-5">gpt-5</option>
                            <option value="gpt-5.1">gpt-5.1</option>
                            <option value="gpt-5-pro">gpt-5-pro</option>
                        </optgroup>
                        <optgroup label="O-Series (Reasoning)">
                            <option value="o1-mini">o1-mini</option>
                            <option value="o3-mini">o3-mini</option>
                            <option value="o4-mini">o4-mini</option>
                        </optgroup>
                    </select>
                </div>
                <div class="form-group">
                    <label class="form-label">Premium Model (Complex/Deep)</label>
                    <select class="form-control" id="premiumModel">
                        <optgroup label="Confirmed Working">
                            <option value="gpt-4o">gpt-4o (Recommended)</option>
                            <option value="gpt-4o-mini">gpt-4o-mini</option>
                            <option value="gpt-4-turbo">gpt-4-turbo</option>
                            <option value="gpt-4o-2024-08-06">gpt-4o-2024-08-06</option>
                            <option value="chatgpt-4o-latest">chatgpt-4o-latest</option>
                        </optgroup>
                        <optgroup label="GPT-4.1 Series">
                            <option value="gpt-4.1">gpt-4.1</option>
                            <option value="gpt-4.1-mini">gpt-4.1-mini</option>
                            <option value="gpt-4.1-nano">gpt-4.1-nano</option>
                        </optgroup>
                        <optgroup label="GPT-5 Series">
                            <option value="gpt-5.1">gpt-5.1 (Flagship)</option>
                            <option value="gpt-5">gpt-5</option>
                            <option value="gpt-5-pro">gpt-5-pro</option>
                            <option value="gpt-5-mini">gpt-5-mini</option>
                            <option value="gpt-5-nano">gpt-5-nano</option>
                        </optgroup>
                        <optgroup label="O-Series (Reasoning)">
                            <option value="o3">o3 (Advanced)</option>
                            <option value="o3-mini">o3-mini</option>
                            <option value="o4-mini">o4-mini</option>
                            <option value="o1">o1</option>
                            <option value="o1-mini">o1-mini</option>
                            <option value="o1-pro">o1-pro</option>
                        </optgroup>
                    </select>
                </div>
            </div>

            <div class="form-group">
                <label class="form-label">System Prompt Template</label>
                <textarea class="form-control textarea-editor" id="systemPrompt" spellcheck="false"></textarea>
                <div class="stat-subtitle" style="margin-top: 8px;">
                    Use <code>%s</code> as a placeholder for the current date/time.
                </div>
            </div>
        </div>

        <div id="toast" class="toast"></div>

        <div class="section">
            <div class="section-header">
                <div class="section-title">
                    📋 Request Logs
                </div>
                <div class="filters">
                    <select id="filterModel">
                        <option value="">All Models</option>
                        <option value="gpt-4o-mini">Mini (cheap)</option>
                        <option value="gpt-5.1">Full (gpt-5.1)</option>
                    </select>
                    <select id="filterStatus">
                        <option value="">All Status</option>
                        <option value="success">Success</option>
                        <option value="error">Error</option>
                    </select>
                    <input type="date" id="filterStartDate" title="Start Date">
                    <input type="date" id="filterEndDate" title="End Date">
                    <button class="btn btn-secondary" onclick="clearFilters()">Clear</button>
                    <button class="btn" onclick="loadLogs()">Apply</button>
                </div>
            </div>

            <div id="logsContainer">
                <div class="loading">
                    <div class="spinner"></div>
                </div>
            </div>

            <div class="pagination" id="pagination" style="display: none;">
                <div class="pagination-info" id="paginationInfo">
                    Showing 0 of 0 entries
                </div>
                <div class="pagination-buttons">
                    <button class="btn btn-secondary" id="prevPage" onclick="prevPage()">← Previous</button>
                    <button class="btn btn-secondary" id="nextPage" onclick="nextPage()">Next →</button>
                </div>
            </div>
        </div>
    </div>

    <script>
        // CSRF token for protecting state-changing requests
        const csrfToken = '{{CSRF_TOKEN}}';
        
        let currentOffset = 0;
        const limit = 50;
        let totalEntries = 0;
        let autoRefreshInterval = null;
        let expandedRows = new Set();

        // Initialize
        document.addEventListener('DOMContentLoaded', () => {
            loadStats();
            loadLogs();
            loadConfig();
            setupAutoRefresh();
        });

        function setupAutoRefresh() {
            const checkbox = document.getElementById('autoRefresh');
            
            const startRefresh = () => {
                if (autoRefreshInterval) clearInterval(autoRefreshInterval);
                autoRefreshInterval = setInterval(() => {
                    loadStats();
                    if (currentOffset === 0) loadLogs(); // Only refresh if on first page
                }, 10000);
            };

            const stopRefresh = () => {
                if (autoRefreshInterval) {
                    clearInterval(autoRefreshInterval);
                    autoRefreshInterval = null;
                }
            };

            checkbox.addEventListener('change', (e) => {
                if (e.target.checked) {
                    startRefresh();
                } else {
                    stopRefresh();
                }
            });

            // Start auto-refresh by default
            startRefresh();
        }

        async function loadStats() {
            try {
                const response = await fetch('/admin/stats');
                const stats = await response.json();
                
                document.getElementById('requestsToday').textContent = stats.total_requests_today.toLocaleString();
                document.getElementById('totalRequests').textContent = stats.total_requests.toLocaleString();
                document.getElementById('avgResponseTime').textContent = stats.avg_response_time_ms.toFixed(0);
                document.getElementById('errorRate').textContent = stats.error_rate.toFixed(2) + '%';
                document.getElementById('modelNano').textContent = stats.model_usage.nano.toLocaleString();
                document.getElementById('modelFull').textContent = stats.model_usage.full.toLocaleString();
                document.getElementById('uptime').textContent = 'Up: ' + stats.uptime;
            } catch (error) {
                console.error('Failed to load stats:', error);
            }
        }

        async function loadLogs() {
            const container = document.getElementById('logsContainer');
            container.innerHTML = '<div class="loading"><div class="spinner"></div></div>';

            try {
                const params = new URLSearchParams({
                    limit: limit.toString(),
                    offset: currentOffset.toString()
                });

                const model = document.getElementById('filterModel').value;
                const status = document.getElementById('filterStatus').value;
                const startDate = document.getElementById('filterStartDate').value;
                const endDate = document.getElementById('filterEndDate').value;

                if (model) params.append('model', model);
                if (status) params.append('status', status);
                if (startDate) params.append('start_date', startDate);
                if (endDate) params.append('end_date', endDate);

                const response = await fetch('/admin/logs?' + params);
                const data = await response.json();
                
                totalEntries = data.total;
                renderLogs(data.entries);
                updatePagination(data);
            } catch (error) {
                console.error('Failed to load logs:', error);
                container.innerHTML = '<div class="empty-state"><div class="empty-state-icon">❌</div><div>Failed to load logs</div></div>';
            }
        }

        function renderLogs(entries) {
            const container = document.getElementById('logsContainer');
            
            if (!entries || entries.length === 0) {
                container.innerHTML = '<div class="empty-state"><div class="empty-state-icon">📭</div><div>No logs found</div></div>';
                return;
            }

            const table = document.createElement('table');
            table.className = 'logs-table';
            
            let html = ` + "`" + `
                <thead>
                    <tr>
                        <th>Time</th>
                        <th>Request ID</th>
                        <th>Model</th>
                        <th>Response Time</th>
                        <th>Status</th>
                        <th>Details</th>
                    </tr>
                </thead>
                <tbody>
            ` + "`" + `;

            entries.forEach((entry, index) => {
                const isExpanded = expandedRows.has(entry.id);
                const hasContent = entry.input || entry.output;
                const safeId = escapeHtml(entry.id);
                
                html += ` + "`" + `
                    <tr class="${isExpanded ? 'expanded' : ''}" data-id="${safeId}">
                        <td>${formatTime(entry.timestamp)}</td>
                        <td class="request-id">${safeId}</td>
                        <td>
                            <span class="badge ${entry.model && entry.model.includes('mini') ? 'badge-nano' : 'badge-full'}">
                                ${entry.model && entry.model.includes('mini') ? 'Mini' : 'Full'}
                            </span>
                        </td>
                        <td>${entry.response_time_ms}ms</td>
                        <td>
                            <span class="badge ${entry.status === 'success' ? 'badge-success' : 'badge-error'}">
                                ${entry.status}
                            </span>
                            ${entry.error_message ? ` + "`" + `<br><small style="color: var(--accent-red)">${escapeHtml(entry.error_message)}</small>` + "`" + ` : ''}
                        </td>
                        <td>
                            ${hasContent ? ` + "`" + `
                                <button class="expand-btn ${isExpanded ? 'active' : ''}" onclick="toggleDetails('${safeId.replace(/'/g, "\\'")}')">
                                    ${isExpanded ? '▼ Hide' : '▶ View'}
                                </button>
                            ` + "`" + ` : '<span style="color: var(--text-secondary); font-size: 11px;">No data</span>'}
                        </td>
                    </tr>
                    <tr class="detail-row ${isExpanded ? 'visible' : ''}" id="detail-${safeId}">
                        <td colspan="6" class="detail-cell">
                            <div class="detail-content">
                                ${entry.input ? ` + "`" + `
                                    <div class="detail-section">
                                        <div class="detail-label">💬 Input (Question)</div>
                                        <div class="detail-text input">${escapeHtml(entry.input)}</div>
                                    </div>
                                ` + "`" + ` : ''}
                                ${entry.output ? ` + "`" + `
                                    <div class="detail-section">
                                        <div class="detail-label">🤖 Output (Response)</div>
                                        <div class="detail-text output">${escapeHtml(entry.output)}</div>
                                    </div>
                                ` + "`" + ` : ''}
                            </div>
                        </td>
                    </tr>
                ` + "`" + `;
            });

            html += '</tbody>';
            table.innerHTML = html;

            container.innerHTML = '';
            container.appendChild(table);
        }

        function toggleDetails(id) {
            const detailRow = document.getElementById('detail-' + id);
            const mainRow = document.querySelector(` + "`" + `tr[data-id="${id}"]` + "`" + `);
            const btn = mainRow.querySelector('.expand-btn');
            
            if (expandedRows.has(id)) {
                expandedRows.delete(id);
                detailRow.classList.remove('visible');
                mainRow.classList.remove('expanded');
                btn.classList.remove('active');
                btn.textContent = '▶ View';
            } else {
                expandedRows.add(id);
                detailRow.classList.add('visible');
                mainRow.classList.add('expanded');
                btn.classList.add('active');
                btn.textContent = '▼ Hide';
            }
        }

        function formatTime(timestamp) {
            const date = new Date(timestamp);
            return date.toLocaleString('pt-BR', {
                day: '2-digit',
                month: '2-digit',
                hour: '2-digit',
                minute: '2-digit',
                second: '2-digit'
            });
        }

        function escapeHtml(text) {
            if (!text) return '';
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }

        function updatePagination(data) {
            const pagination = document.getElementById('pagination');
            const info = document.getElementById('paginationInfo');
            const prevBtn = document.getElementById('prevPage');
            const nextBtn = document.getElementById('nextPage');

            pagination.style.display = 'flex';
            
            const start = data.offset + 1;
            const end = data.offset + data.count;
            info.textContent = ` + "`" + `Showing ${start}-${end} of ${data.total} entries` + "`" + `;

            prevBtn.disabled = currentOffset === 0;
            nextBtn.disabled = currentOffset + data.count >= data.total;
        }

        function prevPage() {
            currentOffset = Math.max(0, currentOffset - limit);
            expandedRows.clear();
            loadLogs();
        }

        function nextPage() {
            currentOffset += limit;
            expandedRows.clear();
            loadLogs();
        }

        function clearFilters() {
            document.getElementById('filterModel').value = '';
            document.getElementById('filterStatus').value = '';
            document.getElementById('filterStartDate').value = '';
            document.getElementById('filterEndDate').value = '';
            currentOffset = 0;
            expandedRows.clear();
            loadLogs();
        }

        // Config Management
        async function loadConfig() {
            try {
                const response = await fetch('/admin/config');
                if (!response.ok) throw new Error('Failed to load config');
                
                const config = await response.json();
                
                if (config.system_prompt) {
                    document.getElementById('systemPrompt').value = config.system_prompt;
                }
                if (config.standard_model) {
                    document.getElementById('standardModel').value = config.standard_model;
                }
                if (config.premium_model) {
                    document.getElementById('premiumModel').value = config.premium_model;
                }
            } catch (error) {
                console.error('Error loading config:', error);
                // Don't show error toast on load to avoid annoyance if backend isn't ready
            }
        }

        async function saveConfig() {
            const btn = document.getElementById('saveConfigBtn');
            const btnText = btn.querySelector('.btn-text');
            const spinner = btn.querySelector('.spinner');
            
            // Lock UI
            btn.disabled = true;
            btnText.style.display = 'none';
            spinner.style.display = 'block';
            
            const config = {
                system_prompt: document.getElementById('systemPrompt').value,
                standard_model: document.getElementById('standardModel').value,
                premium_model: document.getElementById('premiumModel').value
            };
            
            try {
                const response = await fetch('/admin/config', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'X-CSRF-Token': csrfToken
                    },
                    body: JSON.stringify(config)
                });
                
                if (!response.ok) throw new Error('Failed to save config');
                
                showToast('Configuration saved successfully', 'success');
            } catch (error) {
                console.error('Error saving config:', error);
                showToast('Failed to save configuration', 'error');
            } finally {
                // Unlock UI
                btn.disabled = false;
                btnText.style.display = 'block';
                spinner.style.display = 'none';
            }
        }

        function showToast(message, type) {
            const toast = document.getElementById('toast');
            toast.textContent = message;
            toast.className = 'toast toast-' + type + ' show';
            
            setTimeout(() => {
                toast.classList.remove('show');
            }, 3000);
        }
    </script>
</body>
</html>`
