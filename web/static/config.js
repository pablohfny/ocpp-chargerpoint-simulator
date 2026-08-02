// Config tab: every runtime-editable setting in one screen, persisted to a JSON
// file by the backend. The environment only seeds the defaults on a fresh boot.
Sim.tabs.config = (function () {
    // RUNTIME_LABELS names the boot-fixed values shown read-only.
    const RUNTIME_LABELS = {
        serverAddr: 'Servidor OCPP',
        clientId: 'Client ID',
        httpPort: 'Porta HTTP',
        authEnabled: 'Basic auth',
        settingsPath: 'Arquivo de configuração',
        partnersPath: 'Arquivo de parceiros'
    };

    function init() {
        document.getElementById('settingsForm').addEventListener('submit', save);
        document.getElementById('reloadSettingsButton').addEventListener('click', load);
    }

    function activate() {
        load();
    }

    // The Config tab is a form, so polling would fight the operator's typing.
    function refresh() {}

    async function load() {
        try {
            const payload = await Sim.request(Sim.API + '/settings');
            render(payload);
            Sim.setMessage('settingsMessage', '');
        } catch (error) {
            Sim.setMessage('settingsMessage', error.message, 'error');
        }
    }

    function render(payload) {
        const settings = payload.settings || {};
        document.getElementById('settingOcpiBaseUrl').value = settings.ocpiBaseUrl || '';
        document.getElementById('settingPublicBaseUrl').value = settings.publicBaseUrl || '';
        document.getElementById('settingLocation').value = settings.defaultLocationId || '';
        document.getElementById('settingEvse').value = settings.defaultEvseUid || '';
        document.getElementById('settingConnector').value = settings.defaultConnectorId || '';
        document.getElementById('settingBattery').value = settings.batteryCapacityKwh || '';

        renderRuntime(payload.runtime || {});
    }

    function renderRuntime(runtime) {
        const container = document.getElementById('runtimeList');
        Sim.clear(container);

        Object.keys(RUNTIME_LABELS).forEach((key) => {
            const raw = runtime[key];
            if (raw === undefined || raw === null || raw === '') return;

            const item = Sim.element('div', 'readonly-item');
            item.appendChild(Sim.element('div', 'stat-label', RUNTIME_LABELS[key]));
            item.appendChild(Sim.element('div', 'value',
                typeof raw === 'boolean' ? (raw ? 'habilitado' : 'desabilitado') : String(raw)));
            container.appendChild(item);
        });
    }

    async function save(event) {
        event.preventDefault();

        const button = document.getElementById('saveSettingsButton');
        button.disabled = true;
        Sim.setMessage('settingsMessage', 'Salvando...');

        try {
            const payload = await Sim.request(Sim.API + '/settings', {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    ocpiBaseUrl: document.getElementById('settingOcpiBaseUrl').value.trim(),
                    publicBaseUrl: document.getElementById('settingPublicBaseUrl').value.trim(),
                    defaultLocationId: document.getElementById('settingLocation').value.trim(),
                    defaultEvseUid: document.getElementById('settingEvse').value.trim(),
                    defaultConnectorId: document.getElementById('settingConnector').value.trim(),
                    batteryCapacityKwh: Number(document.getElementById('settingBattery').value)
                })
            });
            render(payload);
            Sim.setMessage('settingsMessage', 'Configuração salva e persistida em arquivo.', 'success');
        } catch (error) {
            Sim.setMessage('settingsMessage', error.message, 'error');
        } finally {
            button.disabled = false;
        }
    }

    return { init: init, activate: activate, refresh: refresh };
})();
