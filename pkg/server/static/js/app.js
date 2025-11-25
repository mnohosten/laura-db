// LauraDB Admin Console JavaScript

// API Base URL (assuming same origin)
const API_BASE = '';

// Current state
let currentCollection = '';
let collections = [];

// Initialize the application
document.addEventListener('DOMContentLoaded', () => {
    initializeTabs();
    initializeModals();
    initializeConsole();
    initializeCollections();
    initializeDocuments();
    initializeIndexes();
    initializeStatistics();
    loadServerInfo();
    loadCollectionsList();
});

// Tab Navigation
function initializeTabs() {
    const tabs = document.querySelectorAll('.nav-tab');
    const tabContents = document.querySelectorAll('.tab-content');

    tabs.forEach(tab => {
        tab.addEventListener('click', () => {
            const tabName = tab.dataset.tab;

            // Update active tab
            tabs.forEach(t => t.classList.remove('active'));
            tab.classList.add('active');

            // Update active content
            tabContents.forEach(content => {
                content.classList.remove('active');
            });
            document.getElementById(`${tabName}-tab`).classList.add('active');

            // Load tab-specific data
            if (tabName === 'statistics') {
                loadStatistics();
            }
        });
    });
}

// Modal Management
function initializeModals() {
    const modals = document.querySelectorAll('.modal');
    const closeButtons = document.querySelectorAll('.modal-close');

    closeButtons.forEach(btn => {
        btn.addEventListener('click', () => {
            modals.forEach(modal => modal.classList.remove('active'));
        });
    });

    // Close modal on background click
    modals.forEach(modal => {
        modal.addEventListener('click', (e) => {
            if (e.target === modal) {
                modal.classList.remove('active');
            }
        });
    });
}

// Load Server Info
async function loadServerInfo() {
    try {
        const response = await fetch(`${API_BASE}/_health`);
        const data = await response.json();

        if (data.ok) {
            document.getElementById('server-uptime').textContent = `Uptime: ${data.result.uptime}`;
            document.getElementById('server-status').textContent = data.result.status === 'healthy' ? 'Connected' : 'Disconnected';
            document.getElementById('server-status').className = data.result.status === 'healthy' ? 'status-badge' : 'status-badge error';
        }
    } catch (error) {
        console.error('Failed to load server info:', error);
        document.getElementById('server-status').textContent = 'Error';
        document.getElementById('server-status').className = 'status-badge error';
    }
}

// Load Collections List
async function loadCollectionsList() {
    try {
        const response = await fetch(`${API_BASE}/_collections`);
        const data = await response.json();

        if (data.ok) {
            collections = data.result.collections || [];
            updateCollectionSelects();
        }
    } catch (error) {
        console.error('Failed to load collections:', error);
    }
}

// Update Collection Select Dropdowns
function updateCollectionSelects() {
    const selects = [
        document.getElementById('collection-select'),
        document.getElementById('doc-collection-select'),
        document.getElementById('idx-collection-select')
    ];

    selects.forEach(select => {
        const currentValue = select.value;
        select.innerHTML = '<option value="">-- Select Collection --</option>';
        collections.forEach(col => {
            const option = document.createElement('option');
            option.value = col;
            option.textContent = col;
            if (col === currentValue) option.selected = true;
            select.appendChild(option);
        });
    });
}

// Console Tab
function initializeConsole() {
    const executeBtn = document.getElementById('execute-query');
    const clearBtn = document.getElementById('clear-query');
    const queryInput = document.getElementById('query-input');
    const queryOutput = document.getElementById('query-output');
    const collectionSelect = document.getElementById('collection-select');
    const queryType = document.getElementById('query-type');

    executeBtn.addEventListener('click', async () => {
        const collection = collectionSelect.value;
        const operation = queryType.value;
        const query = queryInput.value.trim();

        if (!collection) {
            showOutput({ error: 'Please select a collection' });
            return;
        }

        try {
            const queryData = query ? JSON.parse(query) : {};
            let result;

            switch (operation) {
                case 'find':
                    result = await executeFind(collection, queryData);
                    break;
                case 'insert':
                    result = await executeInsert(collection, queryData);
                    break;
                case 'update':
                    result = await executeUpdate(collection, queryData);
                    break;
                case 'delete':
                    result = await executeDelete(collection, queryData);
                    break;
                case 'aggregate':
                    result = await executeAggregate(collection, queryData);
                    break;
                default:
                    result = { error: 'Unknown operation' };
            }

            showOutput(result);
        } catch (error) {
            showOutput({ error: error.message });
        }
    });

    clearBtn.addEventListener('click', () => {
        queryInput.value = '';
        queryOutput.textContent = '';
    });

    function showOutput(data) {
        queryOutput.textContent = JSON.stringify(data, null, 2);
    }
}

// Execute Query Operations
async function executeFind(collection, query) {
    const response = await fetch(`${API_BASE}/${collection}/_search`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ query: query })
    });
    return await response.json();
}

async function executeInsert(collection, document) {
    const response = await fetch(`${API_BASE}/${collection}/_doc`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(document)
    });
    return await response.json();
}

async function executeUpdate(collection, update) {
    // This is simplified - in practice you'd need to specify which document to update
    const id = update._id || '';
    delete update._id;

    const response = await fetch(`${API_BASE}/${collection}/_doc/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(update)
    });
    return await response.json();
}

async function executeDelete(collection, query) {
    // This is simplified - assumes query has _id
    const id = query._id || '';
    const response = await fetch(`${API_BASE}/${collection}/_doc/${id}`, {
        method: 'DELETE'
    });
    return await response.json();
}

async function executeAggregate(collection, pipeline) {
    const response = await fetch(`${API_BASE}/${collection}/_aggregate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ pipeline: pipeline })
    });
    return await response.json();
}

// Collections Tab
function initializeCollections() {
    const createBtn = document.getElementById('create-collection-btn');
    const confirmBtn = document.getElementById('confirm-create-collection');
    const modal = document.getElementById('create-collection-modal');
    const nameInput = document.getElementById('new-collection-name');

    createBtn.addEventListener('click', () => {
        modal.classList.add('active');
        nameInput.value = '';
    });

    confirmBtn.addEventListener('click', async () => {
        const name = nameInput.value.trim();
        if (!name) {
            alert('Please enter a collection name');
            return;
        }

        try {
            const response = await fetch(`${API_BASE}/${name}`, {
                method: 'PUT'
            });
            const result = await response.json();

            if (result.ok) {
                modal.classList.remove('active');
                await loadCollectionsList();
                await displayCollections();
            } else {
                alert(`Error: ${result.message}`);
            }
        } catch (error) {
            alert(`Error creating collection: ${error.message}`);
        }
    });

    // Load collections when tab is activated
    const collectionsTab = document.querySelector('[data-tab="collections"]');
    collectionsTab.addEventListener('click', displayCollections);
}

async function displayCollections() {
    const container = document.getElementById('collections-list');
    container.innerHTML = '<div class="loading">Loading collections...</div>';

    try {
        const response = await fetch(`${API_BASE}/_stats`);
        const data = await response.json();

        if (data.ok && data.result.collection_stats) {
            const stats = data.result.collection_stats;
            const collectionsHTML = Object.keys(stats).map(name => {
                const col = stats[name];
                return `
                    <div class="list-item">
                        <div class="list-item-header">
                            <div class="list-item-title">${name}</div>
                            <div class="list-item-actions">
                                <button class="btn btn-danger btn-sm" onclick="deleteCollection('${name}')">Delete</button>
                            </div>
                        </div>
                        <div class="list-item-body">
                            <div class="list-item-meta">
                                <span>Documents: ${col.count}</span>
                                <span>Indexes: ${col.indexes}</span>
                            </div>
                        </div>
                    </div>
                `;
            }).join('');

            container.innerHTML = collectionsHTML || '<div class="empty-state"><h3>No collections found</h3><p>Create a collection to get started</p></div>';
        } else {
            container.innerHTML = '<div class="empty-state"><h3>No collections found</h3></div>';
        }
    } catch (error) {
        container.innerHTML = `<div class="empty-state"><h3>Error loading collections</h3><p>${error.message}</p></div>`;
    }
}

async function deleteCollection(name) {
    if (!confirm(`Are you sure you want to delete collection "${name}"?`)) {
        return;
    }

    try {
        const response = await fetch(`${API_BASE}/${name}`, {
            method: 'DELETE'
        });
        const result = await response.json();

        if (result.ok) {
            await loadCollectionsList();
            await displayCollections();
        } else {
            alert(`Error: ${result.message}`);
        }
    } catch (error) {
        alert(`Error deleting collection: ${error.message}`);
    }
}

// Documents Tab
function initializeDocuments() {
    const loadBtn = document.getElementById('load-documents');
    const collectionSelect = document.getElementById('doc-collection-select');

    loadBtn.addEventListener('click', async () => {
        const collection = collectionSelect.value;
        if (!collection) {
            alert('Please select a collection');
            return;
        }
        await displayDocuments(collection);
    });
}

async function displayDocuments(collection) {
    const container = document.getElementById('documents-list');
    container.innerHTML = '<div class="loading">Loading documents...</div>';

    try {
        const response = await fetch(`${API_BASE}/${collection}/_search`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ query: {} })
        });
        const data = await response.json();

        if (data.ok && data.result) {
            const docs = Array.isArray(data.result) ? data.result : [data.result];
            const docsHTML = docs.map(doc => `
                <div class="document-item">
                    <pre>${JSON.stringify(doc, null, 2)}</pre>
                </div>
            `).join('');

            container.innerHTML = docsHTML || '<div class="empty-state"><h3>No documents found</h3></div>';
        } else {
            container.innerHTML = '<div class="empty-state"><h3>No documents found</h3></div>';
        }
    } catch (error) {
        container.innerHTML = `<div class="empty-state"><h3>Error loading documents</h3><p>${error.message}</p></div>`;
    }
}

// Indexes Tab
function initializeIndexes() {
    const loadBtn = document.getElementById('load-indexes');
    const createBtn = document.getElementById('create-index-btn');
    const confirmBtn = document.getElementById('confirm-create-index');
    const collectionSelect = document.getElementById('idx-collection-select');
    const modal = document.getElementById('create-index-modal');

    loadBtn.addEventListener('click', async () => {
        const collection = collectionSelect.value;
        if (!collection) {
            alert('Please select a collection');
            return;
        }
        await displayIndexes(collection);
    });

    createBtn.addEventListener('click', () => {
        const collection = collectionSelect.value;
        if (!collection) {
            alert('Please select a collection first');
            return;
        }
        modal.classList.add('active');
    });

    confirmBtn.addEventListener('click', async () => {
        const collection = collectionSelect.value;
        const name = document.getElementById('index-name').value.trim();
        const field = document.getElementById('index-field').value.trim();
        const unique = document.getElementById('index-unique').checked;

        if (!name || !field) {
            alert('Please fill in all fields');
            return;
        }

        try {
            const response = await fetch(`${API_BASE}/${collection}/_index`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    name: name,
                    field: field,
                    unique: unique
                })
            });
            const result = await response.json();

            if (result.ok) {
                modal.classList.remove('active');
                await displayIndexes(collection);
            } else {
                alert(`Error: ${result.message}`);
            }
        } catch (error) {
            alert(`Error creating index: ${error.message}`);
        }
    });
}

async function displayIndexes(collection) {
    const container = document.getElementById('indexes-list');
    container.innerHTML = '<div class="loading">Loading indexes...</div>';

    try {
        const response = await fetch(`${API_BASE}/${collection}/_index`);
        const data = await response.json();

        if (data.ok && data.result) {
            const indexes = Array.isArray(data.result) ? data.result : [data.result];
            const indexesHTML = indexes.map(idx => `
                <div class="list-item">
                    <div class="list-item-header">
                        <div class="list-item-title">${idx.name || 'Unnamed'}</div>
                        <div class="list-item-actions">
                            <button class="btn btn-danger btn-sm" onclick="deleteIndex('${collection}', '${idx.name}')">Delete</button>
                        </div>
                    </div>
                    <div class="list-item-body">
                        <div class="list-item-meta">
                            <span>Field: ${idx.field || idx.fields?.join(', ') || 'N/A'}</span>
                            ${idx.unique ? '<span>Unique</span>' : ''}
                            ${idx.type ? `<span>Type: ${idx.type}</span>` : ''}
                        </div>
                    </div>
                </div>
            `).join('');

            container.innerHTML = indexesHTML || '<div class="empty-state"><h3>No indexes found</h3></div>';
        } else {
            container.innerHTML = '<div class="empty-state"><h3>No indexes found</h3></div>';
        }
    } catch (error) {
        container.innerHTML = `<div class="empty-state"><h3>Error loading indexes</h3><p>${error.message}</p></div>`;
    }
}

async function deleteIndex(collection, name) {
    if (!confirm(`Are you sure you want to delete index "${name}"?`)) {
        return;
    }

    try {
        const response = await fetch(`${API_BASE}/${collection}/_index/${name}`, {
            method: 'DELETE'
        });
        const result = await response.json();

        if (result.ok) {
            await displayIndexes(collection);
        } else {
            alert(`Error: ${result.message}`);
        }
    } catch (error) {
        alert(`Error deleting index: ${error.message}`);
    }
}

// Statistics Tab
function initializeStatistics() {
    const refreshBtn = document.getElementById('refresh-stats');
    refreshBtn.addEventListener('click', loadStatistics);
}

async function loadStatistics() {
    const container = document.getElementById('stats-content');
    container.innerHTML = '<div class="loading">Loading statistics...</div>';

    try {
        const response = await fetch(`${API_BASE}/_stats`);
        const data = await response.json();

        if (data.ok && data.result) {
            const stats = data.result;
            let html = `
                <div class="stat-card">
                    <h3>Database Overview</h3>
                    <div class="stat-details">
                        <div class="stat-row">
                            <span class="stat-row-label">Database Name</span>
                            <span class="stat-row-value">${stats.name || 'default'}</span>
                        </div>
                        <div class="stat-row">
                            <span class="stat-row-label">Collections</span>
                            <span class="stat-row-value">${stats.collections || 0}</span>
                        </div>
                        <div class="stat-row">
                            <span class="stat-row-label">Active Transactions</span>
                            <span class="stat-row-value">${stats.active_transactions || 0}</span>
                        </div>
                    </div>
                </div>
            `;

            if (stats.collection_stats) {
                Object.keys(stats.collection_stats).forEach(name => {
                    const col = stats.collection_stats[name];
                    html += `
                        <div class="stat-card">
                            <h3>Collection: ${name}</h3>
                            <div class="stat-value">${col.count}</div>
                            <div class="stat-details">
                                <div class="stat-row">
                                    <span class="stat-row-label">Documents</span>
                                    <span class="stat-row-value">${col.count}</span>
                                </div>
                                <div class="stat-row">
                                    <span class="stat-row-label">Indexes</span>
                                    <span class="stat-row-value">${col.indexes}</span>
                                </div>
                            </div>
                        </div>
                    `;
                });
            }

            container.innerHTML = html;
        } else {
            container.innerHTML = '<div class="empty-state"><h3>No statistics available</h3></div>';
        }
    } catch (error) {
        container.innerHTML = `<div class="empty-state"><h3>Error loading statistics</h3><p>${error.message}</p></div>`;
    }
}

// Update server info periodically
setInterval(loadServerInfo, 10000); // Every 10 seconds
