<template>
  <div class="app-container">
    <!-- Navbar -->
    <header class="navbar">
      <div class="brand">
        <span class="brand-icon">💰</span>
        <span class="brand-name">Meu Dinheiro</span>
      </div>

      <div class="nav-actions">
        <template v-if="currentUser">
          <div class="user-badge">
            <span class="user-avatar">👤</span>
            <span class="user-name">{{ currentUser.name }}</span>
          </div>
          <button class="btn btn-secondary btn-sm" @click="handleLogout">Sair</button>
        </template>
        <template v-else>
          <button class="btn btn-primary" @click="openLoginModal">
            <span class="tg-icon">✈️</span> Entrar com Telegram
          </button>
        </template>
      </div>
    </header>

    <!-- Main Content -->
    <main class="content">
      <!-- Logged-in Dashboard View -->
      <section v-if="currentUser" class="dashboard-section">
        <div class="welcome-banner">
          <h1>Painel Financeiro</h1>
          <p>Sessão vinculada à conta Telegram: <strong>#{{ currentUser.telegram_id }}</strong></p>
        </div>

        <div class="kpi-grid">
          <div class="kpi-card">
            <span class="kpi-label">Saldo em Contas</span>
            <span class="kpi-value text-green">R$ 4.250,00</span>
            <span class="kpi-sub">Atualizado hoje</span>
          </div>
          <div class="kpi-card">
            <span class="kpi-label">Meta de Poupança</span>
            <span class="kpi-value text-blue">R$ 1.500,00</span>
            <span class="kpi-sub">Protegido da fatura</span>
          </div>
          <div class="kpi-card highlight">
            <span class="kpi-label">Saldo Seguro Diário (S2S)</span>
            <span class="kpi-value text-gold">R$ 91,66 / dia</span>
            <span class="kpi-sub">Para os próximos 30 dias</span>
          </div>
          <div class="kpi-card">
            <span class="kpi-label">Status do Ciclo</span>
            <span class="badge badge-success">SAUDÁVEL</span>
            <span class="kpi-sub">Sem risco de déficit</span>
          </div>
        </div>

        <div class="quick-actions">
          <button class="btn btn-outline" @click="handleAction('Nova Transação')">➕ Registrar Gasto</button>
          <button class="btn btn-outline" @click="handleAction('Simular Parcelamento')">🔮 Simular Compra Futura</button>
          <button class="btn btn-outline" @click="handleAction('Check-in Diário')">✅ Fazer Check-in Diário</button>
        </div>
      </section>

      <!-- Public Landing Page View -->
      <section v-else class="hero-section">
        <div class="badge hero-badge">✨ NOVO MODELO FINANCEIRO DETERMINÍSTICO</div>
        <h1 class="hero-title">
          Finanças Pessoais sem Ansiedade:<br/>
          Descubra seu <span class="highlight-text">Saldo Seguro Diário</span>.
        </h1>
        <p class="hero-description">
          Pare de tentar adivinhar se você pode comprar algo hoje. O Meu Dinheiro calcula o valor exato
          que você pode gastar a cada dia sem nunca comprometer suas contas fixas ou sua meta de reserva.
        </p>

        <div class="hero-cta">
          <button class="btn btn-primary btn-lg" @click="openLoginModal">
            <span class="tg-icon">✈️</span> Acessar com Telegram
          </button>
          <a href="#features" class="btn btn-ghost btn-lg">Como funciona ↓</a>
        </div>

        <!-- Features Grid -->
        <div id="features" class="features-grid">
          <div class="feature-card">
            <div class="feature-icon">🛡️</div>
            <h3>S2S Diário</h3>
            <p>Calcula dinamicamente a liquidez livre dividida pelos dias restantes do ciclo financeiro.</p>
          </div>
          <div class="feature-card">
            <div class="feature-icon">🔮</div>
            <h3>Simulador What-If</h3>
            <p>Projete compras parceladas em até 12 ciclos futuros antes de passar o cartão.</p>
          </div>
          <div class="feature-card">
            <div class="feature-icon">✈️</div>
            <h3>Zero Senhas</h3>
            <p>Autenticação nativa com Telegram ID em 1 clique, com proteção de sessão e cookies seguros.</p>
          </div>
          <div class="feature-card">
            <div class="feature-icon">⚡</div>
            <h3>Alta Performance</h3>
            <p>Back-end reescrito em Go com precisão decimal bancária e locking transacional atômico.</p>
          </div>
        </div>
      </section>
    </main>

    <!-- Telegram Auth Modal -->
    <div v-if="showModal" class="modal-backdrop" @click.self="closeModal">
      <div class="modal-card">
        <button class="modal-close" @click="closeModal">✕</button>
        
        <div v-if="modalLoading" class="modal-body text-center">
          <div class="spinner"></div>
          <p>Gerando chave de desafio segura...</p>
        </div>

        <div v-else-if="challenge" class="modal-body text-center">
          <div class="modal-header-icon">✈️</div>
          <h2>Conecte seu Telegram</h2>
          <p class="modal-instruction">
            Para entrar sem senhas, confirme seu acesso no bot oficial:
          </p>

          <a :href="challenge.deep_link" target="_blank" class="btn btn-primary btn-block tg-btn">
            Abrir Telegram (@{{ botUsername }})
          </a>

          <div class="polling-indicator">
            <div class="radar-dot"></div>
            <span>Aguardando autorização no Telegram...</span>
          </div>

          <div class="fallback-box">
            <span class="fallback-label">Ou envie manualmente para o bot:</span>
            <code class="fallback-code">/start auth_{{ challenge.token }}</code>
          </div>

          <p class="modal-timer">Válido por mais {{ remainingSeconds }} segundos</p>
        </div>

        <div v-else-if="modalError" class="modal-body text-center">
          <div class="modal-header-icon error">⚠️</div>
          <h2>Erro de Conexão</h2>
          <p class="text-muted">{{ modalError }}</p>
          <button class="btn btn-secondary btn-block" @click="openLoginModal">Tentar Novamente</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'

const currentUser = ref(null)
const showModal = ref(false)
const modalLoading = ref(false)
const modalError = ref(null)
const challenge = ref(null)
const botUsername = ref('meu_dinheiro_bot')
const remainingSeconds = ref(300)

let pollInterval = null
let countdownInterval = null

onMounted(async () => {
  await checkSession()
})

onUnmounted(() => {
  stopPolling()
})

async function checkSession() {
  try {
    const res = await fetch('/auth/me', { credentials: 'include' })
    if (res.ok) {
      const data = await res.json()
      currentUser.value = data
    }
  } catch (err) {
    // Session not found, remains public
  }
}

async function openLoginModal() {
  showModal.value = true
  modalLoading.value = true
  modalError.value = null
  challenge.value = null

  try {
    const res = await fetch('/auth/telegram/challenge', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
    })

    if (!res.ok) {
      throw new Error(`Falha ao gerar desafio (HTTP ${res.status})`)
    }

    const data = await res.json()
    challenge.value = data
    remainingSeconds.value = 300
    modalLoading.value = false

    startPolling(data.token)
    startCountdown()
  } catch (err) {
    modalLoading.value = false
    modalError.value = err.message || 'Erro ao conectar com servidor'
  }
}

function startPolling(token) {
  stopPolling()
  pollInterval = setInterval(async () => {
    try {
      const res = await fetch(`/auth/telegram/poll?token=${token}`, { credentials: 'include' })
      if (res.ok) {
        const data = await res.json()
        if (data.status === 'authorized' && data.user) {
          currentUser.value = data.user
          closeModal()
        }
      }
    } catch (err) {
      console.warn('Polling error:', err)
    }
  }, 2000)
}

function startCountdown() {
  clearInterval(countdownInterval)
  countdownInterval = setInterval(() => {
    if (remainingSeconds.value > 0) {
      remainingSeconds.value--
    } else {
      closeModal()
    }
  }, 1000)
}

function stopPolling() {
  if (pollInterval) {
    clearInterval(pollInterval)
    pollInterval = null
  }
  if (countdownInterval) {
    clearInterval(countdownInterval)
    countdownInterval = null
  }
}

function closeModal() {
  showModal.value = false
  stopPolling()
}

async function handleLogout() {
  try {
    await fetch('/auth/logout', { method: 'POST', credentials: 'include' })
  } finally {
    currentUser.value = null
  }
}

function handleAction(name) {
  alert(`${name}: Funcionalidade ativa na API! Use o bot no Telegram ou endpoints REST.`)
}
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
  font-family: 'Plus Jakarta Sans', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}

body {
  background-color: #0b0f19;
  color: #f3f4f6;
  min-height: 100vh;
}

.app-container {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

/* Navbar */
.navbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.25rem 2.5rem;
  border-bottom: 1px solid #1f293d;
  background: rgba(11, 15, 25, 0.8);
  backdrop-filter: blur(12px);
  position: sticky;
  top: 0;
  z-index: 50;
}

.brand {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.brand-icon {
  font-size: 1.75rem;
}

.brand-name {
  font-size: 1.25rem;
  font-weight: 700;
  color: #fff;
  letter-spacing: -0.5px;
}

.nav-actions {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.user-badge {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  background: #1e293b;
  padding: 0.4rem 0.8rem;
  border-radius: 9999px;
  font-size: 0.875rem;
}

/* Buttons */
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 0.65rem 1.25rem;
  border-radius: 8px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
  border: none;
  text-decoration: none;
  font-size: 0.95rem;
}

.btn-primary {
  background: #2563eb;
  color: #fff;
}

.btn-primary:hover {
  background: #1d4ed8;
  transform: translateY(-1px);
}

.btn-secondary {
  background: #334155;
  color: #f1f5f9;
}

.btn-secondary:hover {
  background: #475569;
}

.btn-outline {
  background: transparent;
  border: 1px solid #334155;
  color: #e2e8f0;
}

.btn-outline:hover {
  background: #1e293b;
  border-color: #475569;
}

.btn-ghost {
  background: transparent;
  color: #94a3b8;
}

.btn-ghost:hover {
  color: #fff;
}

.btn-lg {
  padding: 0.85rem 1.75rem;
  font-size: 1.05rem;
}

.btn-sm {
  padding: 0.4rem 0.75rem;
  font-size: 0.85rem;
}

.btn-block {
  width: 100%;
}

/* Content */
.content {
  flex: 1;
  max-width: 1100px;
  margin: 0 auto;
  padding: 3rem 1.5rem;
  width: 100%;
}

/* Hero */
.hero-section {
  text-align: center;
  padding: 4rem 0 2rem;
}

.hero-badge {
  display: inline-block;
  background: rgba(37, 99, 235, 0.15);
  color: #60a5fa;
  border: 1px solid rgba(37, 99, 235, 0.3);
  padding: 0.35rem 0.9rem;
  border-radius: 9999px;
  font-size: 0.8rem;
  font-weight: 700;
  margin-bottom: 1.5rem;
}

.hero-title {
  font-size: 2.75rem;
  line-height: 1.2;
  font-weight: 800;
  letter-spacing: -1px;
  margin-bottom: 1.25rem;
}

.highlight-text {
  background: linear-gradient(135deg, #38bdf8 0%, #818cf8 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.hero-description {
  font-size: 1.15rem;
  line-height: 1.6;
  color: #94a3b8;
  max-width: 680px;
  margin: 0 auto 2.5rem;
}

.hero-cta {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 1rem;
  margin-bottom: 4.5rem;
}

/* Features */
.features-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 1.5rem;
  text-align: left;
}

.feature-card {
  background: #111827;
  border: 1px solid #1f2937;
  padding: 1.75rem;
  border-radius: 12px;
  transition: transform 0.2s ease, border-color 0.2s ease;
}

.feature-card:hover {
  transform: translateY(-3px);
  border-color: #374151;
}

.feature-icon {
  font-size: 2rem;
  margin-bottom: 1rem;
}

.feature-card h3 {
  font-size: 1.15rem;
  font-weight: 700;
  margin-bottom: 0.5rem;
  color: #fff;
}

.feature-card p {
  color: #9ca3af;
  font-size: 0.9rem;
  line-height: 1.5;
}

/* Dashboard View */
.dashboard-section {
  padding: 1rem 0;
}

.welcome-banner {
  margin-bottom: 2rem;
}

.welcome-banner h1 {
  font-size: 2rem;
  font-weight: 700;
  margin-bottom: 0.25rem;
}

.welcome-banner p {
  color: #94a3b8;
}

.kpi-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 1.25rem;
  margin-bottom: 2rem;
}

.kpi-card {
  background: #111827;
  border: 1px solid #1f2937;
  border-radius: 12px;
  padding: 1.5rem;
  display: flex;
  flex-direction: column;
}

.kpi-card.highlight {
  border-color: #2563eb;
  background: linear-gradient(180deg, #111827 0%, #172554 100%);
}

.kpi-label {
  font-size: 0.85rem;
  color: #94a3b8;
  font-weight: 600;
  margin-bottom: 0.5rem;
}

.kpi-value {
  font-size: 1.75rem;
  font-weight: 800;
  margin-bottom: 0.5rem;
}

.kpi-sub {
  font-size: 0.75rem;
  color: #64748b;
}

.text-green { color: #34d399; }
.text-blue { color: #60a5fa; }
.text-gold { color: #fbbf24; }

.badge {
  display: inline-block;
  padding: 0.25rem 0.6rem;
  border-radius: 6px;
  font-size: 0.75rem;
  font-weight: 700;
  width: fit-content;
}

.badge-success {
  background: rgba(52, 211, 153, 0.15);
  color: #34d399;
}

.quick-actions {
  display: flex;
  gap: 1rem;
  flex-wrap: wrap;
}

/* Modal */
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.75);
  backdrop-filter: blur(6px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
  padding: 1rem;
}

.modal-card {
  background: #111827;
  border: 1px solid #1f2937;
  border-radius: 16px;
  width: 100%;
  max-width: 440px;
  padding: 2rem;
  position: relative;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
}

.modal-close {
  position: absolute;
  top: 1rem;
  right: 1rem;
  background: transparent;
  border: none;
  color: #64748b;
  font-size: 1.25rem;
  cursor: pointer;
}

.modal-close:hover {
  color: #fff;
}

.modal-header-icon {
  font-size: 2.5rem;
  margin-bottom: 1rem;
}

.modal-card h2 {
  font-size: 1.4rem;
  font-weight: 700;
  margin-bottom: 0.5rem;
}

.modal-instruction {
  color: #94a3b8;
  font-size: 0.9rem;
  margin-bottom: 1.5rem;
}

.tg-btn {
  background: #0284c7;
}

.tg-btn:hover {
  background: #0369a1;
}

.polling-indicator {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.6rem;
  margin: 1.5rem 0 1rem;
  color: #94a3b8;
  font-size: 0.85rem;
}

.radar-dot {
  width: 8px;
  height: 8px;
  background: #38bdf8;
  border-radius: 50%;
  box-shadow: 0 0 0 0 rgba(56, 189, 248, 0.7);
  animation: pulse 1.5s infinite;
}

@keyframes pulse {
  0% {
    transform: scale(0.95);
    box-shadow: 0 0 0 0 rgba(56, 189, 248, 0.7);
  }
  70% {
    transform: scale(1);
    box-shadow: 0 0 0 8px rgba(56, 189, 248, 0);
  }
  100% {
    transform: scale(0.95);
    box-shadow: 0 0 0 0 rgba(56, 189, 248, 0);
  }
}

.fallback-box {
  background: #0b0f19;
  border: 1px solid #1f2937;
  border-radius: 8px;
  padding: 0.75rem;
  margin-top: 1rem;
  font-size: 0.8rem;
}

.fallback-label {
  display: block;
  color: #64748b;
  margin-bottom: 0.35rem;
}

.fallback-code {
  color: #38bdf8;
  font-family: monospace;
  word-break: break-all;
}

.modal-timer {
  color: #64748b;
  font-size: 0.75rem;
  margin-top: 1rem;
}

.spinner {
  width: 36px;
  height: 36px;
  border: 3px solid #1f2937;
  border-top-color: #2563eb;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  margin: 2rem auto 1rem;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.text-center { text-align: center; }
.text-muted { color: #94a3b8; }
</style>
