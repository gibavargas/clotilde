// CSRF token for protecting state-changing requests
// Token is read from data-csrf-token attribute on the script tag
const csrfToken = (() => {
    const script = document.querySelector('script[data-csrf-token]');
    return script ? script.dataset.csrfToken : '';
})();

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
    
    let html = `
        <thead>
            <tr>
                <th>Time</th>
                <th>Request ID</th>
                <th>Model</th>
                <th>Category</th>
                <th>Response Time</th>
                <th>Status</th>
                <th>Details</th>
            </tr>
        </thead>
        <tbody>
    `;

    entries.forEach((entry, index) => {
        const isExpanded = expandedRows.has(entry.id);
        const hasContent = entry.input || entry.output;
        const safeId = escapeHtml(entry.id);
        
        html += `
            <tr class="${isExpanded ? 'expanded' : ''}" data-id="${safeId}">
                <td>${formatTime(entry.timestamp)}</td>
                <td class="request-id">${safeId}</td>
                <td>
                    ${entry.model ? `
                        <span class="badge badge-model ${entry.model.includes('mini') || entry.model.includes('nano') ? 'badge-nano' : 'badge-full'}" title="${escapeHtml(entry.model)}">
                            ${escapeHtml(entry.model)}
                        </span>
                    ` : '<span style="color: var(--text-secondary); font-size: 11px;">N/A</span>'}
                </td>
                <td>
                    ${entry.category ? `
                        <span class="badge badge-model" style="background: rgba(88, 166, 255, 0.15); color: var(--accent-cyan);">
                            ${formatCategory(entry.category)}
                        </span>
                    ` : '<span style="color: var(--text-secondary); font-size: 11px;">-</span>'}
                </td>
                <td>${entry.response_time_ms}ms</td>
                <td>
                    <span class="badge ${entry.status === 'success' ? 'badge-success' : 'badge-error'}">
                        ${entry.status}
                    </span>
                    ${entry.error_message ? `<br><small style="color: var(--accent-red)">${escapeHtml(entry.error_message)}</small>` : ''}
                </td>
                <td>
                    ${hasContent ? `
                        <button class="expand-btn ${isExpanded ? 'active' : ''}" onclick="toggleDetails('${safeId.replace(/'/g, "\\'")}')">
                            ${isExpanded ? '▼ Hide' : '▶ View'}
                        </button>
                    ` : '<span style="color: var(--text-secondary); font-size: 11px;">No data</span>'}
                </td>
            </tr>
            <tr class="detail-row ${isExpanded ? 'visible' : ''}" id="detail-${safeId}">
                <td colspan="7" class="detail-cell">
                    <div class="detail-content">
                        ${entry.input ? `
                            <div class="detail-section">
                                <div class="detail-label">💬 Input (Question)</div>
                                <div class="detail-text input">${escapeHtml(entry.input)}</div>
                            </div>
                        ` : ''}
                        ${entry.output ? `
                            <div class="detail-section">
                                <div class="detail-label">🤖 Output (Response)</div>
                                <div class="detail-text output">${escapeHtml(entry.output)}</div>
                            </div>
                        ` : ''}
                    </div>
                </td>
            </tr>
        `;
    });

    html += '</tbody>';
    table.innerHTML = html;

    container.innerHTML = '';
    container.appendChild(table);
}

function toggleDetails(id) {
    const detailRow = document.getElementById('detail-' + id);
    const mainRow = document.querySelector(`tr[data-id="${id}"]`);
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

function formatCategory(category) {
    if (!category) return '-';
    // Convert snake_case to Title Case
    return category.split('_').map(word => 
        word.charAt(0).toUpperCase() + word.slice(1)
    ).join(' ');
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
    info.textContent = `Showing ${start}-${end} of ${data.total} entries`;

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
        
        // Load base system prompt (prefer base_system_prompt, fallback to system_prompt for legacy)
        const basePrompt = config.base_system_prompt || config.system_prompt || '';
        if (basePrompt) {
            document.getElementById('baseSystemPrompt').value = basePrompt;
            // Legacy support
            document.getElementById('systemPrompt').value = basePrompt;
        }
        
        // Load category prompts
        if (config.category_prompts) {
            if (config.category_prompts.web_search) {
                document.getElementById('categoryPromptWebSearch').value = config.category_prompts.web_search;
            }
            if (config.category_prompts.complex) {
                document.getElementById('categoryPromptComplex').value = config.category_prompts.complex;
            }
            if (config.category_prompts.factual) {
                document.getElementById('categoryPromptFactual').value = config.category_prompts.factual;
            }
            if (config.category_prompts.mathematical) {
                document.getElementById('categoryPromptMathematical').value = config.category_prompts.mathematical;
            }
            if (config.category_prompts.creative) {
                document.getElementById('categoryPromptCreative').value = config.category_prompts.creative;
            }
        }
        
        if (config.standard_model) {
            document.getElementById('standardModel').value = config.standard_model;
        }
        if (config.premium_model) {
            document.getElementById('premiumModel').value = config.premium_model;
        }
        if (config.perplexity_enabled !== undefined) {
            document.getElementById('perplexityEnabled').checked = config.perplexity_enabled;
        } else {
            // Default to true if not set
            document.getElementById('perplexityEnabled').checked = true;
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
    
    // Build config with base prompt and category prompts
    const basePrompt = document.getElementById('baseSystemPrompt').value;
    const categoryPrompts = {};
    
    const webSearchPrompt = document.getElementById('categoryPromptWebSearch').value.trim();
    if (webSearchPrompt) categoryPrompts.web_search = webSearchPrompt;
    
    const complexPrompt = document.getElementById('categoryPromptComplex').value.trim();
    if (complexPrompt) categoryPrompts.complex = complexPrompt;
    
    const factualPrompt = document.getElementById('categoryPromptFactual').value.trim();
    if (factualPrompt) categoryPrompts.factual = factualPrompt;
    
    const mathematicalPrompt = document.getElementById('categoryPromptMathematical').value.trim();
    if (mathematicalPrompt) categoryPrompts.mathematical = mathematicalPrompt;
    
    const creativePrompt = document.getElementById('categoryPromptCreative').value.trim();
    if (creativePrompt) categoryPrompts.creative = creativePrompt;
    
    const config = {
        base_system_prompt: basePrompt,
        category_prompts: categoryPrompts,
        standard_model: document.getElementById('standardModel').value,
        premium_model: document.getElementById('premiumModel').value,
        perplexity_enabled: document.getElementById('perplexityEnabled').checked,
        // Legacy support
        system_prompt: basePrompt
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

