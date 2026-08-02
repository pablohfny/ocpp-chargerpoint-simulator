// Carregador tab: the virtual charge point on its own, driven entirely by the
// existing /api/v1 connector actions and OCPP triggers.
Sim.tabs.carregador = (function () {
    const CONNECTOR_ID = 1;
    const ID_TAG = 'COCKPIT_UI';
    const RING_CIRCUMFERENCE = 2 * Math.PI * 52;

    // Statuses in which a charge is under way, so the primary button stops it.
    const ACTIVE_STATUSES = ['Preparing', 'Charging', 'SuspendedEV', 'SuspendedEVSE'];

    let currentStatus = null;
    let busy = false;

    function init() {
        document.getElementById('chargerConnectorLabel').textContent = 'Conector ' + CONNECTOR_ID;
        document.getElementById('chargerPrimary').addEventListener('click', onPrimaryClick);

        document.querySelectorAll('[data-charger-action]').forEach((button) => {
            button.addEventListener('click', () => runConnectorAction(button.dataset.chargerAction));
        });
        document.querySelectorAll('[data-charger-trigger]').forEach((button) => {
            button.addEventListener('click', () => runTrigger(button.dataset.chargerTrigger));
        });
    }

    function activate() {
        refresh();
    }

    async function refresh() {
        try {
            const connector = await Sim.request(Sim.API + '/status/connectors/' + CONNECTOR_ID);
            render(connector);
        } catch (error) {
            Sim.setMessage('chargerMessage', 'Não foi possível ler o conector.', 'error');
        }
    }

    function render(connector) {
        currentStatus = connector.status;

        const percent = connector.batteryPercent || 0;
        const readout = document.getElementById('batteryPercent');
        Sim.clear(readout);
        readout.appendChild(document.createTextNode(String(percent)));
        readout.appendChild(Sim.element('span', null, '%'));

        const ring = document.getElementById('ringValue');
        ring.style.strokeDasharray = RING_CIRCUMFERENCE;
        ring.style.strokeDashoffset = RING_CIRCUMFERENCE * (1 - percent / 100);
        ring.setAttribute('class', 'ring-value ' + ringModifier(connector.status));

        const pill = document.getElementById('chargerStatus');
        pill.textContent = Sim.statusLabel(connector.status);
        pill.className = 'status-pill status-' + String(connector.status).toLowerCase();

        document.getElementById('chargerEnergy').textContent = Sim.formatEnergy(connector.meterValue);
        document.getElementById('chargerTransaction').textContent = connector.currentTransaction || '--';
        document.getElementById('chargerCable').textContent = connector.cablePlugged ? 'Plugado' : 'Solto';

        renderPrimaryButton(connector.status);
    }

    function ringModifier(status) {
        if (status === 'Faulted') return 'faulted';
        if (status === 'SuspendedEV' || status === 'SuspendedEVSE') return 'suspended';
        if (status === 'Charging') return '';
        return 'idle';
    }

    function renderPrimaryButton(status) {
        const button = document.getElementById('chargerPrimary');

        if (busy) {
            button.disabled = true;
            return;
        }

        if (ACTIVE_STATUSES.includes(status)) {
            button.textContent = 'Parar recarga';
            button.className = 'btn btn-danger';
            button.disabled = false;
            return;
        }

        button.textContent = 'Iniciar recarga';
        button.className = 'btn btn-success';
        // Finishing, Faulted, Reserved and Unavailable cannot start a charge.
        button.disabled = status !== 'Available';
    }

    function onPrimaryClick() {
        if (ACTIVE_STATUSES.includes(currentStatus)) {
            stopCharging();
        } else {
            startCharging();
        }
    }

    // startCharging drives the local flow through the existing API: plug the
    // cable, move to Preparing, authorize the tag, then start the transaction
    // so the CSMS assigns a transaction id.
    async function startCharging() {
        busy = true;
        renderPrimaryButton(currentStatus);
        Sim.setMessage('chargerMessage', 'Iniciando recarga...');

        try {
            await tolerate(Sim.post(Sim.API + '/actions/connectors/' + CONNECTOR_ID + '/plug'));
            await Sim.post(Sim.API + '/actions/connectors/' + CONNECTOR_ID + '/preparing');
            await Sim.post(Sim.API + '/ocpp/trigger/authorize', { connectorId: CONNECTOR_ID, idTag: ID_TAG });
            await Sim.post(Sim.API + '/ocpp/trigger/start-transaction', { connectorId: CONNECTOR_ID, idTag: ID_TAG });
            Sim.setMessage('chargerMessage', 'Recarga solicitada. Aguardando o servidor...', 'success');
        } catch (error) {
            Sim.setMessage('chargerMessage', error.message, 'error');
        } finally {
            busy = false;
            refresh();
        }
    }

    // stopCharging puts the connector in Finishing, which makes the simulator
    // send the OCPP StopTransaction, then releases the cable.
    async function stopCharging() {
        busy = true;
        renderPrimaryButton(currentStatus);
        Sim.setMessage('chargerMessage', 'Parando recarga...');

        try {
            await Sim.post(Sim.API + '/actions/connectors/' + CONNECTOR_ID + '/stop');
            await tolerate(Sim.post(Sim.API + '/actions/connectors/' + CONNECTOR_ID + '/unplug'));
            Sim.setMessage('chargerMessage', 'Recarga encerrada.', 'success');
        } catch (error) {
            Sim.setMessage('chargerMessage', error.message, 'error');
        } finally {
            busy = false;
            refresh();
        }
    }

    // tolerate swallows the errors of steps that are already satisfied, such as
    // plugging a cable that is in place.
    async function tolerate(promise) {
        try {
            return await promise;
        } catch (error) {
            return null;
        }
    }

    async function runConnectorAction(action) {
        Sim.setMessage('chargerControlMessage', 'Executando ' + action + '...');

        try {
            const body = action === 'fault'
                ? { errorCode: 'EVCommunicationError', info: 'falha injetada pela aba Carregador' }
                : {};
            const result = await Sim.post(Sim.API + '/actions/connectors/' + CONNECTOR_ID + '/' + action, body);
            Sim.setMessage('chargerControlMessage', result.message || 'Ação executada.', 'success');
        } catch (error) {
            Sim.setMessage('chargerControlMessage', error.message, 'error');
        } finally {
            refresh();
        }
    }

    async function runTrigger(trigger) {
        Sim.setMessage('chargerControlMessage', 'Disparando ' + trigger + '...');

        try {
            const result = await Sim.post(Sim.API + '/ocpp/trigger/' + trigger, { connectorId: CONNECTOR_ID });
            Sim.setMessage('chargerControlMessage', result.message || 'Mensagem enviada.', 'success');
        } catch (error) {
            Sim.setMessage('chargerControlMessage', error.message, 'error');
        } finally {
            refresh();
        }
    }

    return { init: init, activate: activate, refresh: refresh };
})();
