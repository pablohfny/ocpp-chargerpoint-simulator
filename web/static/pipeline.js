// Pipeline tab: the end-to-end cockpit. Every colour on screen comes from the
// Go orchestrator, which derives it from the partner event feed and the real
// charger state — this file only renders and forwards clicks.
Sim.tabs.pipeline = (function () {
    let state = null;
    let busy = false;
    let lastEventId = 0;
    let renderedEvents = 0;

    function init() {
        document.getElementById('armContextButton').addEventListener('click', armContext);
        document.getElementById('actionBar').addEventListener('click', onActionClick);
    }

    function activate() {
        loadPartners();
        refresh();
    }

    // loadPartners fills the context selector and prefills the defaults.
    async function loadPartners() {
        const select = document.getElementById('contextPartner');

        try {
            const partners = await Sim.request(Sim.OCPI_API + '/partners');
            const previous = select.value;
            Sim.clear(select);

            partners.forEach((partner) => {
                const option = document.createElement('option');
                option.value = partner.slug;
                option.textContent = partner.name + ' (' + partner.slug + ')';
                select.appendChild(option);
            });

            if (partners.length === 0) {
                Sim.setMessage('contextMessage', 'Cadastre um parceiro na aba Parceiros para começar.', 'error');
                return;
            }
            select.value = previous || (state && state.context.partnerSlug) || partners[0].slug;
        } catch (error) {
            Sim.setMessage('contextMessage', error.message, 'error');
        }

        await prefillDefaults();
    }

    // prefillDefaults leaves the operator's own edits alone.
    async function prefillDefaults() {
        try {
            const defaults = await Sim.request(Sim.OCPI_API + '/defaults');
            const location = document.getElementById('contextLocation');
            const evse = document.getElementById('contextEvse');
            if (!location.value) location.value = defaults.locationId || '';
            if (!evse.value) evse.value = defaults.evseUid || '';
        } catch (error) {
            // The defaults are a convenience; the form still works without them.
        }
    }

    async function armContext() {
        const button = document.getElementById('armContextButton');
        button.disabled = true;
        Sim.setMessage('contextMessage', 'Armando contexto...');

        try {
            state = await Sim.post(Sim.API + '/pipeline/start-context', {
                partnerSlug: document.getElementById('contextPartner').value,
                locationId: document.getElementById('contextLocation').value.trim(),
                evseUid: document.getElementById('contextEvse').value.trim(),
                connectorId: document.getElementById('contextConnector').value.trim(),
                ocppConnectorId: Number(document.getElementById('contextOcppConnector').value) || 1
            });
            resetEventFeed();
            Sim.setMessage('contextMessage', 'Contexto armado.', 'success');
            Sim.setMessage('actionMessage', '');
            render();
        } catch (error) {
            Sim.setMessage('contextMessage', error.message, 'error');
        } finally {
            button.disabled = false;
        }
    }

    async function onActionClick(event) {
        const button = event.target.closest('[data-action]');
        if (!button || busy) return;

        const action = button.dataset.action;
        busy = true;
        setActionsDisabled(true);
        Sim.setMessage('actionMessage', 'Executando ' + action + '...');

        try {
            state = await Sim.post(Sim.API + '/pipeline/action', { action: action });
            if (action === 'reset') resetEventFeed();
            Sim.setMessage('actionMessage', state.run && state.run.note ? state.run.note : 'Ação executada.', 'success');
        } catch (error) {
            Sim.setMessage('actionMessage', error.message, 'error');
        } finally {
            busy = false;
            render();
        }
    }

    function setActionsDisabled(disabled) {
        document.querySelectorAll('#actionBar .btn').forEach((button) => {
            button.disabled = disabled;
        });
    }

    async function refresh() {
        if (busy) return;

        try {
            state = await Sim.request(Sim.API + '/pipeline/state');
        } catch (error) {
            return;
        }

        render();
        pollEvents();
    }

    function render() {
        if (!state) return;

        renderContextSummary();
        renderHops();
        renderStage();
        renderActions();
        renderRunFacts();
    }

    function renderContextSummary() {
        const context = state.context || {};
        const summary = document.getElementById('contextSummary');

        if (!context.partnerSlug) {
            summary.textContent = 'Nenhum contexto armado.';
            return;
        }
        summary.textContent = context.partnerSlug + ' · ' + (context.locationId || '?') +
            ' / ' + (context.evseUid || '?') + ' · conector OCPP ' + context.ocppConnectorId;
    }

    function renderHops() {
        const container = document.getElementById('pipelineHops');
        Sim.clear(container);

        (state.hops || []).forEach((hop, index) => {
            if (index > 0) {
                container.appendChild(Sim.element('div', 'hop-arrow', '→'));
            }

            const card = Sim.element('div', 'hop ' + hop.status + (hop.expected ? ' expected' : ''));
            card.appendChild(Sim.element('div', 'hop-label', hop.label));

            if (hop.detail) {
                card.appendChild(Sim.element('div', 'hop-detail', hop.detail));
            }
            if (hop.error) {
                card.appendChild(Sim.element('div', 'hop-error', hop.error + (hop.expected ? ' (esperado)' : '')));
            }
            container.appendChild(card);
        });

        const charger = state.charger;
        document.getElementById('pipelineCharger').textContent = charger
            ? Sim.statusLabel(charger.status) + ' · ' + charger.batteryPercent + '% de bateria'
            : '';
    }

    function renderStage() {
        const banner = document.getElementById('stageBanner');
        banner.className = 'stage-banner' +
            (state.stage === 'done' ? ' done' : '') +
            (state.stage === 'failed' ? ' failed' : '');

        document.getElementById('stageName').textContent = state.stageLabel || '--';
        document.getElementById('stageHint').textContent = state.stageHint || '';
    }

    function renderActions() {
        const bar = document.getElementById('actionBar');
        Sim.clear(bar);

        const actions = state.actions || [];
        if (actions.length === 0) {
            bar.appendChild(Sim.element('span', 'hint', 'Nenhuma ação disponível nesta etapa.'));
            return;
        }

        actions.forEach((action) => {
            let className = 'btn';
            if (action.primary) className += ' btn-primary';
            else if (action.destructive) className += ' btn-warn';

            const button = Sim.element('button', className, action.label);
            button.type = 'button';
            button.dataset.action = action.id;
            button.title = action.hint || '';
            button.disabled = busy;
            bar.appendChild(button);
        });
    }

    function renderRunFacts() {
        const container = document.getElementById('runFacts');
        Sim.clear(container);

        const run = state.run || {};
        const facts = [];
        if (run.variant) facts.push('variação: ' + run.variant);
        if (run.commandUid) facts.push('command_uid: ' + run.commandUid);
        if (run.contractId) facts.push('contract_id: ' + run.contractId);
        if (run.tokenUid) facts.push('token_uid: ' + run.tokenUid);
        if (run.sessionId) facts.push('session_id: ' + run.sessionId);

        facts.forEach((fact) => container.appendChild(Sim.element('span', null, fact)));
    }

    function resetEventFeed() {
        lastEventId = state && state.run ? (state.run.sinceEventId || 0) : 0;
        renderedEvents = 0;
        Sim.emptyState(document.getElementById('pipelineEvents'), 'Aguardando eventos desta rodada...');
    }

    // pollEvents appends only the events that belong to the current run, which
    // is what the orchestrator's cursor already marks.
    async function pollEvents() {
        const slug = state.context && state.context.partnerSlug;
        if (!slug) return;

        if (renderedEvents === 0 && lastEventId === 0 && state.run) {
            lastEventId = state.run.sinceEventId || 0;
        }

        let payload;
        try {
            payload = await Sim.request(Sim.OCPI_API + '/partners/' + slug + '/events?after=' + lastEventId);
        } catch (error) {
            return;
        }

        const list = document.getElementById('pipelineEvents');
        if (payload.events.length === 0) {
            if (renderedEvents === 0) {
                Sim.emptyState(list, 'Aguardando eventos desta rodada...');
            }
            return;
        }

        if (renderedEvents === 0) Sim.clear(list);

        payload.events.forEach((event) => {
            list.insertBefore(Sim.renderEvent(event), list.firstChild);
            renderedEvents++;
        });
        lastEventId = payload.lastId || lastEventId;
    }

    return { init: init, activate: activate, refresh: refresh };
})();
