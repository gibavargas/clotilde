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

        .badge-model {
            text-transform: none;
            font-size: 12px;
            font-family: var(--font-mono);
            max-width: 200px;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
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
                <div class="stat-label">Standard Models</div>
                <div class="stat-value" id="modelStandard">-</div>
                <div class="stat-subtitle">Fast/cheap (Haiku, Mini)</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Premium Models</div>
                <div class="stat-value" id="modelPremium">-</div>
                <div class="stat-subtitle">Powerful (Sonnet, GPT-4o)</div>
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
                        <optgroup label="Claude (Anthropic) - FAST ⚡">
                            <option value="claude-haiku-4-5-20251001">Claude Haiku 4.5 (FASTEST - Recommended for CarPlay)</option>
                            <option value="claude-3-5-haiku-20241022">Claude 3.5 Haiku</option>
                            <option value="claude-3-5-haiku-latest">Claude 3.5 Haiku (Latest)</option>
                        </optgroup>
                        <optgroup label="OpenAI - GPT-4o">
                            <option value="gpt-4o-mini">gpt-4o-mini</option>
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
                        <optgroup label="Claude (Anthropic) - FAST ⚡">
                            <option value="claude-haiku-4-5-20251001">Claude Haiku 4.5 (FASTEST - Recommended for CarPlay)</option>
                            <option value="claude-3-5-sonnet-20241022">Claude 3.5 Sonnet (Balanced)</option>
                            <option value="claude-3-5-sonnet-latest">Claude 3.5 Sonnet (Latest)</option>
                            <option value="claude-sonnet-4-20250514">Claude Sonnet 4 (Most Capable)</option>
                            <option value="claude-3-opus-20240229">Claude 3 Opus (Highest Quality)</option>
                        </optgroup>
                        <optgroup label="OpenAI - GPT-4o">
                            <option value="gpt-4o">gpt-4o</option>
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

            <div class="form-group" style="margin-bottom: 24px;">
                <label class="form-label" style="display: flex; align-items: center; gap: 12px; cursor: pointer;">
                    <input type="checkbox" id="perplexityEnabled" style="width: 20px; height: 20px; cursor: pointer; accent-color: var(--accent-cyan);">
                    <span>Enable Perplexity Search API for Web Search</span>
                </label>
                <div class="stat-subtitle" style="margin-top: 8px; margin-left: 32px;">
                    When enabled, uses Perplexity AI Search API instead of OpenAI's web_search tool for web search queries. Enabled by default.
                </div>
            </div>

            <div class="form-group">
                <label class="form-label">Base System Prompt (Core Principles)</label>
                <textarea class="form-control textarea-editor" id="baseSystemPrompt" spellcheck="false"></textarea>
                <div class="stat-subtitle" style="margin-top: 8px;">
                    Core principles shared by all categories. Use <code>%s</code> as a placeholder for the current date/time.
                </div>
            </div>
            
            <div class="form-group" style="margin-top: 24px;">
                <label class="form-label" style="font-size: 16px; font-weight: 600; margin-bottom: 16px;">Category-Specific Prompts (Optional Overrides)</label>
                <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 16px;">
                    <div class="form-group">
                        <label class="form-label">Web Search (Time-Sensitive)</label>
                        <textarea class="form-control textarea-editor" id="categoryPromptWebSearch" spellcheck="false" style="min-height: 120px;"></textarea>
                    </div>
                    <div class="form-group">
                        <label class="form-label">Complex Analysis</label>
                        <textarea class="form-control textarea-editor" id="categoryPromptComplex" spellcheck="false" style="min-height: 120px;"></textarea>
                    </div>
                    <div class="form-group">
                        <label class="form-label">Factual Lookup</label>
                        <textarea class="form-control textarea-editor" id="categoryPromptFactual" spellcheck="false" style="min-height: 120px;"></textarea>
                    </div>
                    <div class="form-group">
                        <label class="form-label">Mathematical</label>
                        <textarea class="form-control textarea-editor" id="categoryPromptMathematical" spellcheck="false" style="min-height: 120px;"></textarea>
                    </div>
                    <div class="form-group" style="grid-column: 1 / -1;">
                        <label class="form-label">Creative/Open-ended</label>
                        <textarea class="form-control textarea-editor" id="categoryPromptCreative" spellcheck="false" style="min-height: 120px;"></textarea>
                    </div>
                </div>
                <div class="stat-subtitle" style="margin-top: 8px;">
                    Leave empty to use default category prompts. These are appended to the base prompt.
                </div>
            </div>
            
            <!-- Legacy support: hidden field for backward compatibility -->
            <input type="hidden" id="systemPrompt" value="">
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
                        <optgroup label="Claude (Anthropic)">
                            <option value="claude-haiku-4-5-20251001">Claude Haiku 4.5</option>
                            <option value="claude-3-5-haiku-20241022">Claude 3.5 Haiku</option>
                            <option value="claude-3-5-haiku-latest">Claude 3.5 Haiku (Latest)</option>
                            <option value="claude-3-5-sonnet-20241022">Claude 3.5 Sonnet</option>
                            <option value="claude-3-5-sonnet-latest">Claude 3.5 Sonnet (Latest)</option>
                            <option value="claude-sonnet-4-20250514">Claude Sonnet 4</option>
                            <option value="claude-3-opus-20240229">Claude 3 Opus</option>
                        </optgroup>
                        <optgroup label="OpenAI">
                            <option value="gpt-4o-mini">gpt-4o-mini</option>
                            <option value="gpt-4o">gpt-4o</option>
                            <option value="gpt-5.1">gpt-5.1</option>
                            <option value="gpt-5">gpt-5</option>
                        </optgroup>
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

    <script src="/admin/static/dashboard.js" data-csrf-token="{{CSRF_TOKEN}}" nonce="{{NONCE}}"></script>
</body>
</html>`
