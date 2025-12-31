document.addEventListener('DOMContentLoaded', () => {
    // --- WebSocket Implementation ---
    let socket = null;
    let selectedBotId = null;
    let bots = [];
    let botDebugLogs = {}; // logStore: { botId: [array of html entries] }
    let saveTimeout = null;

    // --- Item Cache System ---
    const itemCache = {
        byId: new Map(),
        byName: new Map()
    };

    window.getItem = (query) => {
        if (!query && query !== 0) return null;

        let item = null;
        if (typeof query === 'number') {
            item = itemCache.byId.get(query);
        } else if (typeof query === 'string') {
            const id = itemCache.byName.get(query.toLowerCase());
            if (id !== undefined) item = itemCache.byId.get(id);
        }

        if (!item) {
            console.log(`[Cache Miss] requesting item: ${query}`);
            const payload = typeof query === 'number' ? { id: query } : { name: query };
            if (socket && socket.readyState === WebSocket.OPEN) {
                socket.send(JSON.stringify({ type: 'GET_ITEM', data: payload }));
            }
            return null; // Async fetch triggered
        }
        return item; // Synchronous return if cached
    };

    function cacheItem(item) {
        if (!item) return;
        itemCache.byId.set(item.ID, item);
        if (item.Name) itemCache.byName.set(item.Name.toLowerCase(), item.ID);
    }

    function connectWS() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        socket = new WebSocket(`${protocol}//${window.location.host}/ws`);

        socket.onopen = () => {
            console.log('Connected to VortenixGO Server');
        };

        socket.onmessage = (event) => {
            const msg = JSON.parse(event.data);
            if (msg.type === 'UPDATE_LIST') {
                const newBots = msg.data || [];
                const activeIds = newBots.map(b => b.id);

                const selectedWasRemoved = selectedBotId && !activeIds.includes(selectedBotId);

                Object.keys(botDebugLogs).forEach(id => {
                    if (!activeIds.includes(id)) delete botDebugLogs[id];
                });

                bots = newBots;
                renderBotList();

                if (selectedWasRemoved) {
                    deselectBot();
                } else if (selectedBotId) {
                    const updatedBot = bots.find(b => b.id === selectedBotId);
                    if (updatedBot) updateBotDashboard(updatedBot);
                }
            } else if (msg.type === 'ERROR') {
                alert('Error: ' + msg.data);
            } else if (msg.type === 'DEBUG_LOG') {
                appendDebugLog(msg.data);
            } else if (msg.type === 'ITEMS_DATA') {
                // Bulk cache search results
                if (msg.data && Array.isArray(msg.data)) {
                    msg.data.forEach(item => cacheItem(item));
                }
                renderDBItemList(msg.data);
            } else if (msg.type === 'ITEM_DATA') {
                cacheItem(msg.data);
                renderItemDetail(msg.data);
            } else if (msg.type === 'DATABASE_INFO') {
                renderDatabaseInfo(msg.data);
            }
        };

        socket.onclose = () => {
            console.log('Disconnected from Server. Retrying...');
            setTimeout(connectWS, 2000);
        };
    }

    connectWS();

    // --- UI Selection & Updates ---
    const botListContainer = document.getElementById('bot-list');
    const botCountSpan = document.getElementById('bot-count');
    const botDashboard = document.getElementById('bot-dashboard');
    const botNoSelection = document.querySelector('#bot-detail-panel .no-selection');
    const debugLogsContainer = document.getElementById('debug-logs');

    function renderBotList() {
        botListContainer.innerHTML = '';
        botCountSpan.textContent = bots ? bots.length : 0;

        if (!bots || bots.length === 0) {
            botListContainer.innerHTML = '<div class="empty-state">No bots added</div>';
            return;
        }

        bots.forEach(bot => {
            const el = document.createElement('div');
            el.className = `bot-item ${selectedBotId === bot.id ? 'active' : ''}`;
            const nameToShow = bot.display_name || bot.name;
            const subTitle = (bot.ingame_name && bot.email) ? bot.email : bot.type;

            let badgeClass = 'idle';
            const statusTextStr = bot.status || 'Idle';
            const s = statusTextStr.toLowerCase();

            if (s === 'online') badgeClass = 'online';
            else if (s.includes('connect') || s.includes('get')) badgeClass = 'warning';
            else if (s === 'http_block' || s === 'offline') badgeClass = 'danger';

            el.innerHTML = `
                <div class="info">
                    <strong>${nameToShow}</strong>
                    <div style="font-size: 0.8rem; opacity: 0.7;">${subTitle}</div>
                </div>
                <div class="status-indicator">
                    <span class="status-badge ${badgeClass}">${statusTextStr}</span>
                </div>
            `;
            el.onclick = () => selectBot(bot.id);
            botListContainer.appendChild(el);
        });
    }

    function selectBot(id) {
        selectedBotId = id;
        const bot = bots.find(b => b.id === id);
        if (bot) {
            botNoSelection.classList.add('hidden');
            botDashboard.classList.remove('hidden');
            updateBotDashboard(bot);
            renderBotList();
        }
    }

    function deselectBot() {
        selectedBotId = null;
        botNoSelection.classList.remove('hidden');
        botDashboard.classList.add('hidden');
        document.getElementById('bot-console').innerHTML = '<div class="log-line system">> Console cleared...</div>';
        debugLogsContainer.innerHTML = '';
        renderBotList();
    }

    function updateBotDashboard(bot) {
        const nameToShow = bot.display_name || bot.name;
        document.getElementById('detail-name').textContent = nameToShow;

        refreshDebugView(bot.id);

        let emailLabel = document.getElementById('detail-email-label');
        if (bot.email && bot.ingame_name) {
            if (!emailLabel) {
                emailLabel = document.createElement('div');
                emailLabel.id = 'detail-email-label';
                emailLabel.style.fontSize = '0.8rem';
                emailLabel.style.color = 'var(--text-muted)';
                emailLabel.style.marginTop = '-5px';
                document.getElementById('detail-name').parentNode.insertBefore(emailLabel, document.getElementById('detail-name').nextSibling);
            }
            emailLabel.textContent = bot.email;
            emailLabel.style.display = 'block';
        } else if (emailLabel) {
            emailLabel.style.display = 'none';
        }

        document.getElementById('detail-level-gems').textContent = `Lvl ${bot.level || 0} | ${bot.gems || 0} Gems`;
        document.getElementById('detail-world-pos').textContent = `${bot.world || 'EMPTY'} (${bot.pos_x || 0}, ${bot.pos_y || 0})`;

        document.getElementById('detail-played-age').textContent = `${bot.play_time || '0h'} | ${bot.age || 0}d`;
        document.getElementById('detail-ping').textContent = `${bot.ping || 0} ms`;
        document.getElementById('debug-enet').checked = bot.show_enet || false;

        const glogGroup = document.getElementById('detail-glog-group');
        const glogInput = document.getElementById('detail-glog');
        const proxyInput = document.getElementById('detail-proxy');

        if (bot.type === 'gmail' || bot.type === 'apple') {
            glogGroup.classList.remove('hidden');
            if (document.activeElement !== glogInput) glogInput.value = bot.glog || '';
        } else {
            glogGroup.classList.add('hidden');
        }

        if (document.activeElement !== proxyInput) proxyInput.value = bot.proxy || '';

        const statusText = document.getElementById('detail-status-text');
        const statusDot = document.getElementById('status-dot');
        const btnConnect = document.getElementById('btn-connect');
        const btnDisconnect = document.getElementById('btn-disconnect');

        if (statusText && statusDot) {
            const currentStatus = bot.status || 'Idle';
            statusText.textContent = currentStatus;
            const s = currentStatus.toLowerCase();

            // Connect is almost always enabled to allow "Force Connect/Reconnect"
            btnConnect.disabled = false;
            btnDisconnect.disabled = !bot.connected;

            if (s.includes('connect') || s.includes('get')) {
                statusText.style.color = 'var(--warning)';
                statusDot.className = 'status-dot warning';
                btnDisconnect.disabled = false;
            } else if (s === 'online') {
                statusText.style.color = 'var(--success)';
                statusDot.className = 'status-dot online';
                btnDisconnect.disabled = false;
            } else if (s === 'http_block' || s === 'offline' || s === 'disconnected' || s === 'idle') {
                statusText.style.color = 'var(--danger)';
                statusDot.className = 'status-dot offline';
                btnDisconnect.disabled = true;
                if (s === 'idle') statusText.style.color = 'var(--text-muted)';
            } else {
                statusText.style.color = 'var(--text-muted)';
                statusDot.className = 'status-dot offline';
            }
        }

        const devJson = document.getElementById('dev-internal-json');
        if (devJson) {
            // Create a copy without inventory for display
            const { local, ...rest } = bot;
            const cleanLocal = { ...local };
            delete cleanLocal.inventory;
            const displayBot = { ...rest, local: cleanLocal };
            devJson.textContent = JSON.stringify(displayBot, null, 4);
        }

        renderInventory(bot);
        renderPlayers(bot);
    }

    function renderPlayers(bot) {
        const countEl = document.getElementById('player-count');
        const bodyEl = document.getElementById('player-list-body');

        if (!countEl || !bodyEl) return;

        const players = bot.local.players || [];
        countEl.textContent = players.length;

        bodyEl.innerHTML = '';
        if (players.length === 0) {
            bodyEl.innerHTML = '<tr><td colspan="2" class="text-center" style="padding: 40px; opacity: 0.5;">No players in range</td></tr>';
            return;
        }

        players.forEach(p => {
            const row = document.createElement('tr');
            row.className = 'inv-row';

            const leftCell = document.createElement('td');
            leftCell.className = 'inv-info-cell';
            leftCell.innerHTML = `
                <div style="display: flex; flex-direction: column;">
                    <span class="inv-name" style="color: ${p.is_local ? 'var(--primary)' : 'white'}; font-weight: 500;">
                        ${p.name}
                        ${p.is_local ? '<span style="color: var(--primary); opacity: 0.8; font-size: 0.8em; margin-left: 5px;">(YOU)</span>' : ''}
                        ${p.mod ? '<span style="color: var(--danger); opacity: 0.8; font-size: 0.8em; margin-left: 5px;">[MOD]</span>' : ''}
                    </span>
                    <span style="font-size: 0.75em; opacity: 0.5; margin-top: 2px;">NetID: ${p.netid} • UserID: ${p.userid}</span>
                </div>
            `;

            const rightCell = document.createElement('td');
            rightCell.className = 'inv-action-cell';
            rightCell.style.textAlign = 'right';
            rightCell.innerHTML = `
                <div style="display: flex; flex-direction: column; align-items: flex-end;">
                    <span style="font-size: 0.9em;">${p.country || 'Unknown'}</span>
                    <span style="font-size: 0.75em; opacity: 0.5; margin-top: 2px; font-family: monospace;">
                        X: ${Math.round(p.pos_x || 0)} Y: ${Math.round(p.pos_y || 0)}
                    </span>
                </div>
            `;

            row.appendChild(leftCell);
            row.appendChild(rightCell);
            bodyEl.appendChild(row);
        });
    }

    function renderInventory(bot) {
        const slotsEl = document.getElementById('inv-slots');
        const amountEl = document.getElementById('inv-amount');
        const bodyEl = document.getElementById('inventory-body');

        if (!slotsEl || !bodyEl) return;

        const inv = bot.local.inventory || [];
        const maxSlots = bot.local.inventory_slots || 0;

        slotsEl.innerHTML = `<span class="stat-value">${inv.length}</span><span style="opacity: 0.5; margin: 0 4px;">/</span><span class="stat-value" style="opacity: 0.7;">${maxSlots}</span>`;
        amountEl.innerHTML = `<span class="stat-value" style="color: var(--warning);">${(bot.local.gem_count || 0).toLocaleString()}</span>`;

        bodyEl.innerHTML = '';
        if (inv.length === 0) {
            bodyEl.innerHTML = '<tr><td colspan="2" class="text-center" style="padding: 40px; opacity: 0.5;">Inventory is empty</td></tr>';
            return;
        }

        inv.forEach((item) => {
            const row = document.createElement('tr');
            row.className = 'inv-row';

            // Try to get name and info from cache
            const dbItem = window.getItem(item.id);
            const name = item.name || (dbItem ? dbItem.Name : `Item`);
            const clothingType = dbItem ? dbItem.ClothingType : 0;
            const isWearable = clothingType !== 0;

            // Button Logic
            let buttonsHtml = '';

            // Wear/Unwear (Only if wearable)
            if (isWearable) {
                if (item.is_active) {
                    buttonsHtml += `<button class="inv-list-btn unwear" onclick="window.handleInventoryAction('${bot.id}', ${item.id}, 'UNWEAR')">Unwear</button>`;
                } else {
                    buttonsHtml += `<button class="inv-list-btn wear" onclick="window.handleInventoryAction('${bot.id}', ${item.id}, 'WEAR')">Wear</button>`;
                }
            }

            // Drop & Trash (Skip for Fist/Wrench for safety/aesthetics, though user asked for "every item". 
            // Image shows Trash for Fist but not Drop. I will just render them for standard items.)
            // IDs: 18=Fist, 32=Wrench
            if (item.id !== 18 && item.id !== 32) {
                buttonsHtml += `<button class="inv-list-btn drop" onclick="window.handleInventoryAction('${bot.id}', ${item.id}, 'DROP')">Drop</button>`;
            }
            buttonsHtml += `<button class="inv-list-btn trash" onclick="window.handleInventoryAction('${bot.id}', ${item.id}, 'TRASH')">Trash</button>`;


            row.innerHTML = `
                <td class="inv-info-cell">
                    <span class="inv-count">${item.count}x</span>
                    <span class="inv-sep">|</span>
                    <span class="inv-name">${name} [${item.id}]</span>
                    ${item.is_active ? '<span class="inv-equipped">(Equipped)</span>' : ''}
                </td>
                <td class="inv-action-cell">
                    ${buttonsHtml}
                </td>
            `;
            bodyEl.appendChild(row);
        });
    }

    window.handleInventoryAction = (botId, itemId, action) => {
        if (!socket || socket.readyState !== WebSocket.OPEN) return;

        console.log(`[Inventory] Action ${action} for Item ${itemId} on Bot ${botId}`);
        socket.send(JSON.stringify({
            type: 'BOT_ACTION',
            data: {
                id: botId,
                action: 'INVENTORY_ACTION',
                sub_action: action,
                item_id: itemId
            }
        }));
    };

    // --- Event Listeners (Once) ---
    document.querySelectorAll('.nav-item').forEach(item => {
        item.onclick = () => {
            document.querySelectorAll('.nav-item').forEach(n => n.classList.remove('active'));
            document.querySelectorAll('.view-section').forEach(v => v.classList.remove('active'));
            item.classList.add('active');
            const target = item.dataset.target;
            document.getElementById(target).classList.add('active');

            if (target === 'database-view') {
                socket.send(JSON.stringify({ type: 'GET_DATABASE_INFO' }));
            }
        };
    });

    document.getElementById('remove-bot-btn').onclick = () => {
        if (!selectedBotId) return;
        const idToRemove = selectedBotId;
        socket.send(JSON.stringify({ type: 'REMOVE_BOT', data: { id: idToRemove } }));
        delete botDebugLogs[idToRemove];
        deselectBot();
    };

    document.getElementById('btn-connect').onclick = () => {
        if (selectedBotId) socket.send(JSON.stringify({ type: 'BOT_ACTION', data: { id: selectedBotId, action: 'CONNECT' } }));
    };

    document.getElementById('btn-disconnect').onclick = () => {
        if (selectedBotId) socket.send(JSON.stringify({ type: 'BOT_ACTION', data: { id: selectedBotId, action: 'DISCONNECT' } }));
    };

    document.getElementById('detail-glog').oninput = () => debounceSave();
    document.getElementById('detail-proxy').oninput = () => debounceSave();
    document.getElementById('detail-glog').onblur = () => { if (saveTimeout) clearTimeout(saveTimeout); saveBotConfig(); };
    document.getElementById('detail-proxy').onblur = () => { if (saveTimeout) clearTimeout(saveTimeout); saveBotConfig(); };

    function debounceSave() {
        if (saveTimeout) clearTimeout(saveTimeout);
        saveTimeout = setTimeout(saveBotConfig, 500);
    }

    function saveBotConfig() {
        if (!selectedBotId) return;
        const glog = document.getElementById('detail-glog').value;
        const proxy = document.getElementById('detail-proxy').value;
        socket.send(JSON.stringify({ type: 'UPDATE_BOT_CONFIG', data: { id: selectedBotId, glog, proxy } }));
    }

    document.getElementById('btn-execute-lua').onclick = () => {
        const script = document.getElementById('lua-editor').value;
        if (selectedBotId && script) {
            socket.send(JSON.stringify({ type: 'EXECUTE_LUA', data: { id: selectedBotId, script: script } }));
            const consoleBox = document.getElementById('bot-console');
            const entry = document.createElement('div');
            entry.className = 'log-line';
            entry.textContent = `> Sent script to ${selectedBotId}`;
            consoleBox.appendChild(entry);
            consoleBox.scrollTop = consoleBox.scrollHeight;
        }
    };

    // Modal
    const modal = document.getElementById('add-bot-modal');
    document.getElementById('add-bot-btn').onclick = () => {
        document.querySelector('input[value="legacy"]').checked = true;
        document.querySelectorAll('.radio-card').forEach(card => card.classList.remove('active'));
        document.querySelector('input[value="legacy"]').closest('.radio-card').classList.add('active');
        document.getElementById('field-legacy-name').classList.remove('hidden');
        document.getElementById('field-legacy-pass').classList.remove('hidden');
        document.getElementById('field-token-combined').classList.add('hidden');
        document.getElementById('field-external-password').classList.add('hidden');
        document.getElementById('field-glog').classList.add('hidden');
        modal.classList.add('active');
    };
    document.querySelectorAll('.close-modal').forEach(el => el.onclick = () => modal.classList.remove('active'));

    document.querySelectorAll('input[name="bot-type"]').forEach(radio => {
        radio.onchange = (e) => {
            const type = e.target.value;
            document.querySelectorAll('.radio-card').forEach(card => card.classList.remove('active'));
            e.target.closest('.radio-card').classList.add('active');
            if (type === 'legacy') {
                document.getElementById('field-legacy-name').classList.remove('hidden');
                document.getElementById('field-legacy-pass').classList.remove('hidden');
                document.getElementById('field-token-combined').classList.add('hidden');
                document.getElementById('field-external-password').classList.add('hidden');
                document.getElementById('field-glog').classList.add('hidden');
            } else {
                document.getElementById('field-legacy-name').classList.add('hidden');
                document.getElementById('field-legacy-pass').classList.add('hidden');
                document.getElementById('field-token-combined').classList.remove('hidden');
                document.getElementById('field-external-password').classList.remove('hidden');
                document.getElementById('field-glog').classList.remove('hidden');
            }
        };
    });

    document.getElementById('confirm-add-bot').onclick = () => {
        const type = document.querySelector('input[name="bot-type"]:checked').value;
        let name = "", pass = "", glog = document.getElementById('bot-glog').value;
        const proxy = document.getElementById('bot-proxy-input').value;

        if (type === 'legacy') {
            name = document.getElementById('bot-identity').value;
            pass = document.getElementById('bot-password').value;
        } else {
            const combined = document.getElementById('bot-token-combined').value.trim();
            const externalPass = document.getElementById('bot-external-password').value.trim();
            name = combined.split('|')[0] || type;
            pass = combined;
            // Append external password if provided
            if (externalPass) {
                pass = pass + '|' + externalPass;
            }
        }

        if (!name && !pass) { alert('Mohon isi data bot.'); return; }
        socket.send(JSON.stringify({ type: 'ADD_BOT', data: { type, name, pass, glog, proxy } }));
        modal.classList.remove('active');
        document.getElementById('bot-identity').value = '';
        document.getElementById('bot-password').value = '';
        document.getElementById('bot-token-combined').value = '';
        document.getElementById('bot-external-password').value = '';
        document.getElementById('bot-glog').value = '';
        document.getElementById('bot-proxy-input').value = '';
    };

    document.querySelectorAll('.tab-link').forEach(link => {
        link.onclick = () => {
            document.querySelectorAll('.tab-link').forEach(l => l.classList.remove('active'));
            document.querySelectorAll('.tab-pane').forEach(p => p.classList.remove('active'));
            link.classList.add('active');
            document.getElementById(link.dataset.tab).classList.add('active');
        };
    });

    // --- Debug Helpers ---
    const debugFilterAll = document.getElementById('debug-all');
    const debugFilterHttps = document.getElementById('debug-https');
    const debugAutoScroll = document.getElementById('debug-autoscroll');
    const debugLimitInput = document.getElementById('debug-limit');

    function appendDebugLog(log) {
        if (!botDebugLogs[log.bot_id]) botDebugLogs[log.bot_id] = [];
        const entry = document.createElement('div');
        entry.className = `log-entry ${log.is_error ? 'error' : ''} ${log.category.toLowerCase() === 'https' ? 'https-in' : ''}`;
        if (log.message.includes('POST') || log.message.includes('GET')) {
            entry.classList.remove('https-in');
            entry.classList.add('https-out');
        }
        entry.innerHTML = `<span style="color: #64748b; margin-right: 8px;">[${log.time}]</span><span class="debug-tag">${log.category}</span> ${escapeHtml(log.message)}`;

        botDebugLogs[log.bot_id].push(entry.outerHTML);

        const limit = parseInt(debugLimitInput.value) || 100;
        if (botDebugLogs[log.bot_id].length > limit) {
            botDebugLogs[log.bot_id] = botDebugLogs[log.bot_id].slice(-limit);
        }

        if (selectedBotId === log.bot_id) {
            if (log.category === 'HTTPS' && !debugFilterHttps.checked && !debugFilterAll.checked) return;
            debugLogsContainer.appendChild(entry);

            // Limit DOM entries too for performance
            while (debugLogsContainer.children.length > limit) {
                debugLogsContainer.removeChild(debugLogsContainer.firstChild);
            }

            if (debugAutoScroll.checked) {
                debugLogsContainer.scrollTop = debugLogsContainer.scrollHeight;
            }
        }
    }

    function refreshDebugView(botId) {
        debugLogsContainer.innerHTML = '';
        if (!botDebugLogs[botId] || botDebugLogs[botId].length === 0) {
            debugLogsContainer.innerHTML = '<div class="log-entry system">No logs for this bot.</div>';
            return;
        }

        const limit = parseInt(debugLimitInput.value) || 100;
        const logsToShow = botDebugLogs[botId].slice(-limit);

        logsToShow.forEach(html => {
            const temp = document.createElement('div');
            temp.innerHTML = html;
            const entry = temp.firstChild;
            const tag = entry.querySelector('.debug-tag')?.textContent;
            if (tag === 'HTTPS' && !debugFilterHttps.checked && !debugFilterAll.checked) return;
            debugLogsContainer.appendChild(entry);
        });

        if (debugAutoScroll.checked) {
            debugLogsContainer.scrollTop = debugLogsContainer.scrollHeight;
        }
    }

    function escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML.replace(/\n/g, '<br>');
    }

    document.getElementById('clear-debug').onclick = () => {
        if (selectedBotId) { botDebugLogs[selectedBotId] = []; refreshDebugView(selectedBotId); }
    };

    [debugFilterAll, debugFilterHttps, debugLimitInput].forEach(el => {
        el.onchange = () => { if (selectedBotId) refreshDebugView(selectedBotId); };
    });

    document.getElementById('debug-enet').onchange = (e) => {
        if (selectedBotId) {
            socket.send(JSON.stringify({
                type: 'UPDATE_BOT_CONFIG',
                data: { id: selectedBotId, show_enet: e.target.checked }
            }));
        }
    };

    // --- Database View Implementation ---
    const dbSearchInput = document.getElementById('db-search-input');
    const dbItemList = document.getElementById('db-item-list');
    const dbItemCount = document.getElementById('db-item-count');
    const itemInfoContent = document.getElementById('item-info-content');
    const dbNoSelection = document.querySelector('#item-detail-card .no-selection');

    let dbSearchTimeout = null;

    if (dbSearchInput) {
        dbSearchInput.oninput = () => {
            if (dbSearchTimeout) clearTimeout(dbSearchTimeout);
            const query = dbSearchInput.value.trim();
            if (query.length < 2) {
                dbItemList.innerHTML = '<div class="empty-state">Start typing to search items</div>';
                dbItemCount.textContent = '0';
                return;
            }
            dbSearchTimeout = setTimeout(() => {
                socket.send(JSON.stringify({ type: 'SEARCH_ITEMS', data: { query } }));
            }, 300);
        };
    }

    function renderDBItemList(items) {
        if (!dbItemList) return;
        dbItemList.innerHTML = '';
        dbItemCount.textContent = items ? items.length : 0;

        if (!items || items.length === 0) {
            dbItemList.innerHTML = '<div class="empty-state">No items found</div>';
            return;
        }

        items.forEach(item => {
            const el = document.createElement('div');
            el.className = 'db-item';
            el.innerHTML = `
                <div class="item-info">
                    <span class="item-name">${item.Name}</span>
                    <span class="item-id">ID: ${item.ID}</span>
                </div>
                <i class="fa-solid fa-chevron-right" style="font-size: 0.8rem; opacity: 0.5;"></i>
            `;
            el.onclick = () => {
                document.querySelectorAll('.db-item').forEach(i => i.classList.remove('active'));
                el.classList.add('active');

                const cached = window.getItem(item.ID);
                if (cached) {
                    renderItemDetail(cached);
                } else {
                    // This fallback should rarely happen if search results just populated cache
                    socket.send(JSON.stringify({ type: 'GET_ITEM', data: { id: item.ID } }));
                }
            };
            dbItemList.appendChild(el);
        });
    }

    function renderItemDetail(item) {
        if (!item || !itemInfoContent) return;

        if (dbNoSelection) dbNoSelection.classList.add('hidden');
        itemInfoContent.classList.remove('hidden');

        // Reset scroll position
        const rightPanel = document.querySelector('.db-right-panel');
        if (rightPanel) {
            rightPanel.scrollTop = 0;
        }

        // General Info
        document.getElementById('item-name').textContent = item.Name || "Unknown Item";
        document.getElementById('item-id').textContent = `ID: ${item.ID}`;
        document.getElementById('item-rarity').textContent = item.Rarity || 0;

        let growTime = item.GrowTime || 0;
        let growTimeStr = "";
        if (growTime === 0) growTimeStr = "Instant";
        else {
            const h = Math.floor(growTime / 3600);
            const m = Math.floor((growTime % 3600) / 60);
            const s = growTime % 60;
            if (h > 0) growTimeStr += `${h}h `;
            if (m > 0) growTimeStr += `${m}m `;
            if (s > 0) growTimeStr += `${s}s`;
        }
        document.getElementById('item-grow-time').textContent = growTimeStr || "0s";
        document.getElementById('item-description').textContent = item.Description || "No description available.";

        // Gameplay Section
        document.getElementById('item-action-type').textContent = item.ActionType || 0;
        document.getElementById('item-material').textContent = item.Material || 0;
        document.getElementById('item-collision-type').textContent = item.CollisionType || 0;
        document.getElementById('item-clothing-type').textContent = item.ClothingType || 0;
        document.getElementById('item-health').textContent = item.BlockHealth || 0;
        document.getElementById('item-max-item').textContent = item.MaxItem || 0;
        document.getElementById('item-drop-chance').textContent = item.DropChance || 0;
        document.getElementById('item-rayman').textContent = item.IsRayman === 1 ? "Yes" : "No";

        // Visual Section
        document.getElementById('item-texture').textContent = item.TextureFileName || "-";
        document.getElementById('item-texture-hash').textContent = (item.TextureHash || 0).toString(16).toUpperCase();
        document.getElementById('item-texture-pos').textContent = `${item.TextureX || 0}, ${item.TextureY || 0}`;
        document.getElementById('item-render-type').textContent = item.RenderType || 0;
        document.getElementById('item-visual-effect').textContent = item.VisualEffect || 0;
        document.getElementById('item-file-name').textContent = item.FileName || "-";
        document.getElementById('item-file-hash').textContent = (item.FileHash || 0).toString(16).toUpperCase();

        // Seed/Tree Section
        document.getElementById('item-seed-sprites').textContent = `${item.SeedBaseSprite || 0} / ${item.SeedOverlaySprite || 0}`;
        document.getElementById('item-tree-sprites').textContent = `${item.TreeBaseSprite || 0} / ${item.TreeOverlaySprite || 0}`;

        const baseColor = document.getElementById('item-base-color');
        const overlayColor = document.getElementById('item-overlay-color');

        const hexBase = '#' + (item.BaseColor || 0).toString(16).padStart(8, '0').substring(2);
        const hexOverlay = '#' + (item.OverlayColor || 0).toString(16).padStart(8, '0').substring(2);

        baseColor.textContent = hexBase.toUpperCase();
        baseColor.style.color = hexBase;

        overlayColor.textContent = hexOverlay.toUpperCase();
        overlayColor.style.color = hexOverlay;

        document.getElementById('item-ingredient').textContent = item.Ingredient || 0;
        document.getElementById('item-cooking-ing').textContent = item.CookingIngredient || 0;

        // Pet Section
        document.getElementById('item-pet-name').textContent = item.PetName || "-";
        document.getElementById('item-pet-prefix').textContent = item.PetPrefix || "-";
        document.getElementById('item-pet-suffix').textContent = item.PetSuffix || "-";
        document.getElementById('item-pet-ability').textContent = item.PetAbility || "-";

        // Extra Section
        document.getElementById('item-punch-option').textContent = item.PunchOption || "-";
        document.getElementById('item-stripey').textContent = item.IsStripeyWallpaper === 1 ? "Yes" : "No";
        document.getElementById('item-audio').textContent = item.AudioVolume || 0;
        document.getElementById('item-texture2').textContent = item.TexturePath2 || "-";
        document.getElementById('item-extra-opt').textContent = item.ExtraOptions || "-";
        document.getElementById('item-extra-opt2').textContent = item.ExtraOption2 || "-";

        // Render Flags
        const flagsContainer = document.getElementById('item-flags');
        if (flagsContainer) {
            flagsContainer.innerHTML = '';
            if (item.Flags) {
                Object.entries(item.Flags).forEach(([key, value]) => {
                    const flagEl = document.createElement('div');
                    flagEl.className = `flag-item ${value ? 'enabled' : 'disabled'}`;
                    flagEl.innerHTML = `
                        <i class="fa-solid ${value ? 'fa-check-circle' : 'fa-circle-xmark'}"></i>
                        ${key}
                    `;
                    flagsContainer.appendChild(flagEl);
                });
            }
        }
    }

    function renderDatabaseInfo(info) {
        if (!info) return;
        const statusBadge = document.getElementById('db-status-badge');
        if (statusBadge) {
            statusBadge.textContent = info.loaded ? 'Loaded' : 'Error';
            statusBadge.className = info.loaded ? 'badge success' : 'badge danger';
        }
        const versionEl = document.getElementById('db-version');
        if (versionEl) versionEl.textContent = info.version || 0;

        const countEl = document.getElementById('db-total-items');
        if (countEl) countEl.textContent = (info.item_count || 0).toLocaleString();
    }
});
