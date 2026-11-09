<template>
  <section
    class="hidden lg:flex lg:w-[55%] flex flex-col justify-between p-6 lg:p-8 relative overflow-hidden bg-[#F1F5F9]/70 border-r border-slate-200/80"
  >
    <div
      class="absolute inset-0 bg-[radial-gradient(#cbd5e1_1px,transparent_1px)] [background-size:24px_24px] opacity-45 pointer-events-none"
    ></div>
    <div
      class="absolute -top-32 -left-32 w-96 h-96 rounded-full bg-blue-100/60 blur-3xl pointer-events-none"
    ></div>
    <div
      class="absolute top-1/2 -right-40 w-96 h-96 rounded-full bg-teal-100/40 blur-3xl pointer-events-none"
    ></div>

    <header class="relative z-10 flex flex-wrap items-center justify-between gap-3 mb-4">
      <div class="flex items-center gap-3">
        <div class="flex items-center gap-2.5">
          <div
            class="w-9 h-9 rounded-lg bg-gradient-to-br from-blue-700 to-blue-900 text-white flex items-center justify-center shadow-sm shadow-blue-500/20"
          >
            <svg
              class="w-5 h-5"
              fill="none"
              stroke="currentColor"
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2.2"
              viewBox="0 0 24 24"
            >
              <polygon points="12 2 21 7 21 17 12 22 3 17 3 7 12 2"></polygon>
              <circle cx="12" cy="12" fill="currentColor" r="3"></circle>
              <line x1="12" x2="12" y1="2" y2="7"></line>
              <line x1="12" x2="12" y1="17" y2="22"></line>
            </svg>
          </div>
          <div class="flex flex-col">
            <div class="flex items-center gap-1.5">
              <span class="font-bold text-lg tracking-tight text-slate-900 font-sans"
                >PASTRACK</span
              >
              <span
                class="text-[10px] font-semibold uppercase px-1.5 py-0.5 rounded bg-blue-100 text-blue-700 tracking-wide"
                >Telematics</span
              >
            </div>
            <div class="flex items-center gap-2 text-[11px] text-slate-500 font-mono">
              <span>FLEET OS</span>
              <span>•</span>
              <span>v4.18.2-PROD</span>
            </div>
          </div>
        </div>
      </div>

      <div
        class="flex items-center gap-2.5 px-3 py-1.5 rounded-full bg-white border border-slate-200 shadow-sm text-xs font-mono"
      >
        <span class="relative flex h-2 w-2">
          <span
            class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"
          ></span>
          <span class="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
        </span>
        <span class="text-emerald-700 font-semibold tracking-wide">GRID: OPTIMAL</span>
        <span class="text-slate-300">|</span>
        <span class="text-slate-700 font-medium">99.98%</span>
        <span class="text-slate-400 hidden sm:inline">12ms</span>
      </div>
    </header>

    <div class="relative z-10 my-auto py-2 flex flex-col gap-4">
      <div class="max-w-xl">
        <div
          class="inline-flex items-center gap-2 px-2.5 py-1 rounded-md bg-white border border-slate-200/80 text-slate-700 text-xs font-medium mb-3 shadow-xs"
        >
          <span class="material-symbols-outlined text-blue-600 text-base">hub</span>
          <span class="tracking-wider uppercase text-[11px] font-semibold text-slate-600"
            >Mission Control Telemetry</span
          >
        </div>
        <h2
          class="text-2xl xl:text-3xl font-extrabold text-slate-900 tracking-tight leading-[1.15]"
        >
          Complete Fleet Visibility. <br />
          <span
            class="text-transparent bg-clip-text bg-gradient-to-r from-blue-700 via-blue-600 to-teal-600"
            >Smarter Decisions.</span
          >
        </h2>
        <p class="mt-3 text-xs sm:text-sm text-slate-600 max-w-lg leading-relaxed font-normal">
          Monitor commercial assets, track high-frequency journeys, optimize fuel telemetry, and
          automate dispatch intelligence across a synchronized network.
        </p>
      </div>

      <div
        class="relative w-full rounded-xl bg-white border border-slate-200/90 p-3 shadow-[0_8px_30px_rgb(0,0,0,0.04)] overflow-hidden"
      >
        <div
          class="absolute inset-0 bg-gradient-to-b from-transparent via-blue-50/40 to-transparent h-full animate-[pulse_4s_ease-in-out_infinite] pointer-events-none"
        ></div>

        <div
          class="relative z-10 grid grid-cols-3 gap-2.5 sm:gap-4 pb-3 mb-3 border-b border-slate-100"
        >
          <div class="bg-slate-50 border border-slate-200/70 rounded-lg p-2.5 sm:p-3">
            <div class="text-[11px] font-medium text-slate-500 uppercase tracking-wider">
              Active Assets
            </div>
            <div
              class="font-mono text-base sm:text-lg font-bold text-slate-900 flex items-baseline gap-1 mt-0.5"
            >
              1,842 <span class="text-slate-400 text-xs font-normal">/ 1,850</span>
            </div>
          </div>
          <div class="bg-slate-50 border border-slate-200/70 rounded-lg p-2.5 sm:p-3">
            <div class="text-[11px] font-medium text-slate-500 uppercase tracking-wider">
              Fuel Index YoY
            </div>
            <div
              class="font-mono text-base sm:text-lg font-bold text-emerald-600 flex items-baseline gap-1 mt-0.5"
            >
              +14.2% <span class="material-symbols-outlined text-xs font-bold">trending_up</span>
            </div>
          </div>
          <div class="bg-slate-50 border border-slate-200/70 rounded-lg p-2.5 sm:p-3">
            <div class="text-[11px] font-medium text-slate-500 uppercase tracking-wider">
              Live GPS Pings
            </div>
            <div
              class="font-mono text-base sm:text-lg font-bold text-blue-600 flex items-baseline gap-1 mt-0.5"
            >
              48.2k<span class="text-slate-400 text-xs font-normal">/s</span>
            </div>
          </div>
        </div>

        <div
          class="relative z-10 h-52 w-full rounded-lg bg-[#F8FAFC] border border-slate-200/80 overflow-hidden flex items-center justify-center p-3"
        >
          <svg
            class="absolute inset-0 w-full h-full text-slate-300"
            preserveaspectratio="none"
            viewBox="0 0 500 260"
          >
            <defs>
              <linearGradient id="routeGradientA" x1="0%" x2="100%" y1="0%" y2="100%">
                <stop offset="0%" stop-color="#93c5fd" stop-opacity="0.4"></stop>
                <stop offset="50%" stop-color="#2563eb" stop-opacity="0.9"></stop>
                <stop offset="100%" stop-color="#0d9488" stop-opacity="0.9"></stop>
              </linearGradient>
              <linearGradient id="routeGradientB" x1="0%" x2="100%" y1="0%" y2="0%">
                <stop offset="0%" stop-color="#0d9488" stop-opacity="0.8"></stop>
                <stop offset="100%" stop-color="#3b82f6" stop-opacity="0.7"></stop>
              </linearGradient>
            </defs>

            <path
              d="M 0,65 L 500,65 M 0,130 L 500,130 M 0,195 L 500,195"
              fill="none"
              stroke="#E2E8F0"
              stroke-dasharray="3 3"
              stroke-width="0.8"
            ></path>
            <path
              d="M 125,0 L 125,260 M 250,0 L 250,260 M 375,0 L 375,260"
              fill="none"
              stroke="#E2E8F0"
              stroke-dasharray="3 3"
              stroke-width="0.8"
            ></path>

            <path
              d="M 20,40 Q 150,110 260,70 T 480,180"
              fill="none"
              stroke="#CBD5E1"
              stroke-dasharray="4 6"
              stroke-width="1.2"
            ></path>
            <path
              d="M 60,240 Q 180,160 310,210 T 460,60"
              fill="none"
              stroke="#CBD5E1"
              stroke-dasharray="2 4"
              stroke-width="1.2"
            ></path>

            <path
              d="M 65,190 C 130,130 190,140 270,95 C 330,60 390,75 440,110"
              fill="none"
              stroke="url(#routeGradientA)"
              stroke-linecap="round"
              stroke-width="3"
            ></path>
            <path
              d="M 80,85 C 160,70 230,150 360,165"
              fill="none"
              stroke="url(#routeGradientB)"
              stroke-dasharray="6 4"
              stroke-width="2"
            ></path>
          </svg>

          <div
            class="absolute top-[80px] left-[52%] -translate-x-1/2 -translate-y-1/2 group cursor-pointer z-20"
          >
            <div class="relative flex items-center justify-center">
              <span class="animate-ping absolute h-8 w-8 rounded-full bg-blue-500/25"></span>
              <span
                class="h-4 w-4 rounded-full bg-blue-600 shadow-[0_0_12px_rgba(37,99,235,0.5)] flex items-center justify-center border-2 border-white"
              >
                <span class="h-1.5 w-1.5 rounded-full bg-white"></span>
              </span>
            </div>
            <div
              class="mt-1.5 px-2.5 py-1 rounded-md bg-white border border-slate-200/90 shadow-md backdrop-blur text-left"
            >
              <div class="flex items-center gap-1.5">
                <span class="font-mono text-xs font-bold text-blue-700">TX-409</span>
                <span class="h-1.5 w-1.5 rounded-full bg-emerald-500"></span>
                <span class="font-mono text-xs font-semibold text-slate-800">68 mph</span>
              </div>
              <div class="text-[10px] text-slate-500">I-80 Corridor • Optimal</div>
            </div>
          </div>

          <div class="absolute top-[160px] left-[26%] -translate-x-1/2 -translate-y-1/2 z-20">
            <div class="relative flex items-center justify-center">
              <span
                class="h-3.5 w-3.5 rounded-full bg-teal-600 shadow-[0_0_8px_rgba(13,148,136,0.4)] border-2 border-white"
              ></span>
            </div>
            <div
              class="mt-1 px-2 py-0.5 rounded-md bg-white border border-slate-200/90 shadow text-left"
            >
              <div class="font-mono text-[11px] font-bold text-teal-700">FL-112</div>
              <div class="font-mono text-[10px] text-slate-500">42 mph • Eco-Route</div>
            </div>
          </div>

          <div class="absolute bottom-[28px] right-[14%] z-20">
            <div class="relative flex items-center justify-center">
              <span class="h-3 w-3 rounded-full bg-slate-400 border border-white"></span>
            </div>
            <div
              class="mt-1 px-2 py-0.5 rounded-md bg-white border border-slate-200/90 shadow text-left"
            >
              <div class="font-mono text-[11px] font-medium text-slate-700">MV-880</div>
              <div class="font-mono text-[10px] text-slate-400">0 mph • Depot Standby</div>
            </div>
          </div>

          <div
            class="absolute bottom-2.5 left-2.5 flex items-center gap-1.5 bg-white/95 border border-slate-200 px-2.5 py-1 rounded text-slate-600 font-mono text-[11px] shadow-xs"
          >
            <span class="material-symbols-outlined text-blue-600 text-sm">satellite_alt</span>
            <span>GEO-BAND: S-559 • LAT: 41.8781° N</span>
          </div>

          <div
            class="absolute bottom-2.5 right-2.5 flex items-center gap-1.5 bg-white/95 border border-slate-200 px-2 py-1 rounded text-slate-500 font-mono text-[11px] shadow-xs"
          >
            <span class="text-emerald-600 font-semibold">SYNC ACTIVE</span>
            <span>• 60 FPS</span>
          </div>
        </div>
      </div>
    </div>

    <footer class="relative z-10 pt-4 mt-3 flex flex-col gap-1.5 border-t border-slate-200/70">
      <div class="flex items-center gap-2 text-slate-800 font-semibold text-xs sm:text-sm">
        <span class="material-symbols-outlined text-blue-600 text-base">verified</span>
        <span>Intelligent Fleet &amp; Telematics Architecture</span>
      </div>
      <p class="text-xs text-slate-500 leading-normal max-w-xl">
        Hardware-agnostic OBD-II, J1939 CAN-bus &amp; High-Frequency Iridium Satellite Telemetry
        Integration. ISO 27001 &amp; SOC 2 Type II Certified infrastructure.
      </p>
    </footer>
  </section>
</template>
