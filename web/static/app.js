// Shell of the cockpit: tab routing, shared HTTP helpers and the polling loop
// every tab hooks into. Each tab registers itself on Sim.tabs.
const Sim = {
    API: '/api/v1',
    OCPI_API: '/ocpi/api',
    POLL_INTERVAL: 2000,
    tabs: {},
    activeTab: null
};

// request performs a JSON call and surfaces the API's error message.
Sim.request = async function (url, options) {
    const response = await fetch(url, options);
    const payload = await response.json().catch(() => ({}));
    if (!response.ok) {
        throw new Error((payload.error && payload.error.message) || 'Falha na requisição');
    }
    return payload;
};

// post is the shorthand every action button uses.
Sim.post = function (url, body) {
    return Sim.request(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body || {})
    });
};

// setMessage writes an inline status line, colour coded by kind.
Sim.setMessage = function (elementId, text, kind) {
    const element = document.getElementById(elementId);
    if (!element) return;
    element.textContent = text || '';
    element.className = 'message' + (kind ? ' ' + kind : '');
};

// clear empties a container so it can be re-rendered from scratch.
Sim.clear = function (element) {
    while (element.firstChild) {
        element.removeChild(element.firstChild);
    }
};

// element builds a node with a class and text, which keeps the renderers free
// of innerHTML and therefore free of injection.
Sim.element = function (tag, className, text) {
    const node = document.createElement(tag);
    if (className) node.className = className;
    if (text !== undefined && text !== null) node.textContent = text;
    return node;
};

// emptyState replaces a container's content with a neutral placeholder.
Sim.emptyState = function (element, text) {
    Sim.clear(element);
    element.appendChild(Sim.element('div', 'empty-state', text));
};

Sim.formatEnergy = function (watthours) {
    return ((watthours || 0) / 1000).toFixed(2).replace('.', ',') + ' kWh';
};

// statusLabel translates the OCPP connector status for the UI.
Sim.statusLabel = function (status) {
    const labels = {
        Available: 'Disponível',
        Preparing: 'Preparando',
        Charging: 'Carregando',
        SuspendedEV: 'Pausado (veículo)',
        SuspendedEVSE: 'Pausado (carregador)',
        Finishing: 'Finalizando',
        Reserved: 'Reservado',
        Unavailable: 'Indisponível',
        Faulted: 'Com falha'
    };
    return labels[status] || status || '--';
};

// ---------- Event feed rendering, shared by the Pipeline and Parceiros tabs ----------

Sim.renderEvent = function (event) {
    const container = Sim.element('div', 'event direction-' + event.direction + (event.authFailed ? ' auth-failed' : ''));

    const head = Sim.element('div', 'event-head');
    head.appendChild(Sim.element('span', 'badge badge-' + event.kind, String(event.kind).replace('_', ' ')));
    head.appendChild(Sim.element('span', 'event-time', new Date(event.timestamp).toLocaleTimeString('pt-BR')));

    const path = Sim.element('span', 'event-path', event.method + ' ' + event.path);
    path.title = event.path;
    head.appendChild(path);

    if (event.echoOk !== undefined && event.echoOk !== null) {
        let text = event.echoOk ? 'echo ok' : 'echo falhou';
        if (!event.echoOk && event.echoDiff) {
            text += ' (' + event.echoDiff.join(', ') + ')';
        }
        head.appendChild(Sim.element('span', 'echo ' + (event.echoOk ? 'ok' : 'fail'), text));
    }

    container.appendChild(head);

    const facts = [];
    if (event.statusCode) facts.push('HTTP ' + event.statusCode);
    if (event.sessionId) facts.push('session ' + event.sessionId);
    if (event.commandUid) facts.push('uid ' + event.commandUid);
    if (event.contractId) facts.push('contract_id ' + event.contractId);
    if (event.tokenUid) facts.push('token ' + event.tokenUid);
    if (event.error) facts.push('erro: ' + event.error);

    if (facts.length > 0) {
        container.appendChild(Sim.element('div', 'event-meta', facts.join(' | ')));
    }

    if (event.body !== undefined && event.body !== null) {
        const details = document.createElement('details');
        details.appendChild(Sim.element('summary', null, 'Ver payload'));
        details.appendChild(Sim.element('pre', null, JSON.stringify(event.body, null, 2)));
        container.appendChild(details);
    }

    return container;
};

// ---------- Tabs ----------

// showTab activates one tab, telling the previous one to stand down so only the
// visible tab polls.
Sim.showTab = function (name) {
    if (!Sim.tabs[name]) name = 'pipeline';
    if (Sim.activeTab === name) return;

    Sim.activeTab = name;

    document.querySelectorAll('.tab').forEach((button) => {
        button.classList.toggle('active', button.dataset.tab === name);
    });
    document.querySelectorAll('.panel-view').forEach((view) => {
        view.hidden = view.id !== 'view-' + name;
    });

    const tab = Sim.tabs[name];
    if (tab.activate) tab.activate();
};

// syncConnection keeps the header dot honest about the OCPP WebSocket.
Sim.syncConnection = async function () {
    try {
        const connection = await Sim.request(Sim.API + '/status/connection');
        document.getElementById('connectionDot').className = 'dot' + (connection.connected ? ' on' : '');
        document.getElementById('connectionText').textContent = connection.connected ? 'Conectado' : 'Desconectado';
        document.getElementById('stationId').textContent = connection.clientId + ' → ' + connection.serverAddr;
    } catch (error) {
        document.getElementById('connectionDot').className = 'dot';
        document.getElementById('connectionText').textContent = 'Simulador fora do ar';
    }
};

// boot wires the tabs, restores the tab from the URL fragment and starts the
// single polling loop that refreshes whichever tab is on screen.
Sim.boot = function () {
    document.getElementById('tabs').addEventListener('click', (event) => {
        const button = event.target.closest('.tab');
        if (!button) return;
        window.location.hash = button.dataset.tab;
    });

    window.addEventListener('hashchange', () => {
        Sim.showTab(window.location.hash.replace('#', ''));
    });

    Object.keys(Sim.tabs).forEach((name) => {
        if (Sim.tabs[name].init) Sim.tabs[name].init();
    });

    Sim.showTab(window.location.hash.replace('#', ''));
    Sim.syncConnection();

    setInterval(() => {
        Sim.syncConnection();
        const tab = Sim.tabs[Sim.activeTab];
        if (tab && tab.refresh) tab.refresh();
    }, Sim.POLL_INTERVAL);
};
