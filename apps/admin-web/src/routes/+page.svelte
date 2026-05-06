<script lang="ts">
	import { onMount } from 'svelte';

	type Tab = 'config' | 'backend' | 'tts' | 'preview' | 'help' | 'about';
	type TTSConfig = {
		enabled: boolean;
		roomId: number;
		voiceId: string;
		rate: number;
		volume: number;
		maxQueue: number;
		maxAgeSeconds: number;
		templateText: string;
		templateFreeGift: string;
		templatePaidGift: string;
		templateMember: string;
		templateSuperChat: string;
	};
	type BackendConfig = {
		host: string;
		port: number;
		logLevel: string;
		sessdata: string;
		tts: TTSConfig;
	};

	let activeTab = $state<Tab>('config');
	let roomId = $state('');
	let theme = $state('plain');
	let filterLottery = $state(true);
	let copied = $state(false);
	let backendHost = $state('127.0.0.1');
	let backendPort = $state('5090');
	let backendLogLevel = $state('info');
	let backendSessdata = $state('');
	let backendLoading = $state(false);
	let backendSaving = $state(false);
	let backendError = $state('');
	let backendMessage = $state('');
	let sessdataVisible = $state(false);

	// TTS state
	let ttsEnabled = $state(false);
	let ttsRoomId = $state('');
	let ttsVoiceId = $state('ZH-CN');
	let ttsRate = $state(-2);
	let ttsVolume = $state(100);
	let ttsMaxQueue = $state(20);
	let ttsMaxAgeSeconds = $state(0);
	let ttsTemplateText = $state('{author_name} 说，{content}');
	let ttsTemplateFreeGift = $state('{author_name} 赠送了{num}个{gift_name}');
	let ttsTemplatePaidGift = $state('{author_name} 赠送了{num}个{gift_name}，总价{price}元');
	let ttsTemplateMember = $state('{author_name} 购买了{guard_name}');
	let ttsTemplateSuperChat = $state('{author_name} 发送了{price}元的醒目留言，{content}');
	let ttsSaving = $state(false);
	let ttsError = $state('');
	let ttsMessage = $state('');

	const THEMES = [
		{ value: 'plain', label: '简洁' },
		{ value: 'bubble', label: '气泡(深色)' },
		{ value: 'bubble-light', label: '气泡(浅色)' }
	];
	const LOG_LEVELS = [
		{ value: 'debug', label: 'Debug' },
		{ value: 'info', label: 'Info' },
		{ value: 'warn', label: 'Warn' },
		{ value: 'error', label: 'Error' }
	];
	const TAB_LABELS: Record<Tab, string> = {
		config: '直播间设置',
		backend: '配置',
		tts: '语音播报',
		preview: '预览',
		help: '使用说明',
		about: '关于'
	};

	let obsUrl = $derived.by(() => {
		if (typeof window === 'undefined') return '';
		const id = parseInt(roomId, 10);
		if (!id || id <= 0) return '';
		const params = new URLSearchParams();
		params.set('roomId', String(id));
		if (theme && theme !== 'plain') params.set('theme', theme);
		if (filterLottery) params.set('filterLottery', '1');
		return `${window.location.origin}/obs?${params.toString()}`;
	});

	async function copyUrl() {
		if (!obsUrl) return;
		await navigator.clipboard.writeText(obsUrl);
		copied = true;
		setTimeout(() => (copied = false), 2000);
	}

	onMount(() => {
		void loadBackendConfig();
	});

	async function loadBackendConfig() {
		backendLoading = true;
		backendError = '';
		backendMessage = '';

		try {
			const response = await fetch('/api/config');
			const data = await response.json();
			if (!response.ok) {
				backendError = data.error ?? '读取后端配置失败';
				return;
			}

			applyBackendConfig(data as BackendConfig);
		} catch {
			backendError = '读取后端配置失败';
		} finally {
			backendLoading = false;
		}
	}

	async function saveBackendConfig() {
		backendSaving = true;
		backendError = '';
		backendMessage = '';

		try {
			const response = await fetch('/api/config', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					host: backendHost,
					port: Number(backendPort),
					logLevel: backendLogLevel,
					sessdata: backendSessdata,
					tts: {
						enabled: ttsEnabled,
						roomId: Number(ttsRoomId) || 0,
						voiceId: ttsVoiceId,
						rate: ttsRate,
						volume: ttsVolume,
						maxQueue: ttsMaxQueue,
						maxAgeSeconds: ttsMaxAgeSeconds,
						templateText: ttsTemplateText,
						templateFreeGift: ttsTemplateFreeGift,
						templatePaidGift: ttsTemplatePaidGift,
						templateMember: ttsTemplateMember,
						templateSuperChat: ttsTemplateSuperChat
					}
				})
			});
			const data = await response.json();
			if (!response.ok) {
				backendError = data.error ?? '保存后端配置失败';
				return;
			}

			applyBackendConfig(data as BackendConfig);
			backendMessage = '已保存，请手动刷新浏览器源。';
		} catch {
			backendError = '保存后端配置失败';
		} finally {
			backendSaving = false;
		}
	}

	function applyBackendConfig(data: BackendConfig) {
		backendHost = data.host;
		backendPort = String(data.port);
		backendLogLevel = data.logLevel;
		backendSessdata = data.sessdata;
		if (data.tts) {
			ttsEnabled = data.tts.enabled;
			ttsRoomId = String(data.tts.roomId ?? '');
			ttsVoiceId = data.tts.voiceId ?? 'ZH-CN';
			ttsRate = data.tts.rate ?? -2;
			ttsVolume = data.tts.volume ?? 100;
			ttsMaxQueue = data.tts.maxQueue ?? 20;
			ttsMaxAgeSeconds = data.tts.maxAgeSeconds ?? 0;
			ttsTemplateText = data.tts.templateText ?? '{author_name} 说，{content}';
			ttsTemplateFreeGift = data.tts.templateFreeGift ?? '{author_name} 赠送了{num}个{gift_name}';
			ttsTemplatePaidGift =
				data.tts.templatePaidGift ?? '{author_name} 赠送了{num}个{gift_name}，总价{price}元';
			ttsTemplateMember = data.tts.templateMember ?? '{author_name} 购买了{guard_name}';
			ttsTemplateSuperChat =
				data.tts.templateSuperChat ?? '{author_name} 发送了{price}元的醒目留言，{content}';
		}
	}

	async function saveTTSConfig() {
		ttsSaving = true;
		ttsError = '';
		ttsMessage = '';

		try {
			// Load current config first, then merge TTS changes.
			const getResp = await fetch('/api/config');
			const current = await getResp.json();
			if (!getResp.ok) {
				ttsError = current.error ?? '读取当前配置失败';
				return;
			}

			const payload: BackendConfig = {
				...current,
				tts: {
					enabled: ttsEnabled,
					roomId: Number(ttsRoomId) || 0,
					voiceId: ttsVoiceId,
					rate: ttsRate,
					volume: ttsVolume,
					maxQueue: ttsMaxQueue,
					maxAgeSeconds: ttsMaxAgeSeconds,
					templateText: ttsTemplateText,
					templateFreeGift: ttsTemplateFreeGift,
					templatePaidGift: ttsTemplatePaidGift,
					templateMember: ttsTemplateMember,
					templateSuperChat: ttsTemplateSuperChat
				}
			};

			const postResp = await fetch('/api/config', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(payload)
			});
			const data = await postResp.json();
			if (!postResp.ok) {
				ttsError = data.error ?? '保存失败';
				return;
			}

			applyBackendConfig(data as BackendConfig);
			ttsMessage = '已保存，语音设置已热更新（无需重启）。';
		} catch {
			ttsError = '保存失败';
		} finally {
			ttsSaving = false;
		}
	}
</script>

<div class="min-h-screen bg-[#1e1e1e] text-[#e3e3e3]">
	<div class="mx-auto flex max-w-2xl justify-center border-b border-[#3a3a3a]">
		{#each ['config', 'backend', 'tts', 'preview', 'help', 'about'] as Tab[] as tab}
			<button
				class="px-6 py-3 text-sm font-medium transition-colors {activeTab === tab
					? 'border-b-4 border-blue-400 text-white'
					: 'hover:secondary text-[#888]'}"
				onclick={() => (activeTab = tab)}
			>
				{TAB_LABELS[tab]}
			</button>
		{/each}
	</div>

	{#if activeTab === 'config'}
		<div class="tab-panel">
			<h2 class="secondary mb-6 text-base font-semibold">OBS 浏览器源设置</h2>

			<div class="space-y-5">
				<div>
					<label class="primary mb-1.5 block text-sm" for="roomId">直播间号</label>
					<input
						id="roomId"
						type="text"
						placeholder="例如: 213"
						bind:value={roomId}
						class="bg-color w-full rounded border border-[#3a3a3a] px-3 py-2 text-sm text-white placeholder-[#666] focus:border-blue-500 focus:outline-none"
					/>
				</div>

				<div>
					<p class="primary mb-1.5 text-sm">主题</p>
					<div class="flex gap-2">
						{#each THEMES as t}
							<button
								class="rounded border px-3 py-1.5 text-sm transition-colors {theme === t.value
									? 'border-blue-500 bg-blue-500/20 text-blue-300'
									: 'bg-color primary border-[#3a3a3a] hover:border-[#666]'}"
								onclick={() => (theme = t.value)}
							>
								{t.label}
							</button>
						{/each}
					</div>
				</div>

				<div>
					<p class="primary mb-1.5 text-sm">过滤选项</p>
					<label class="secondary flex cursor-pointer items-center gap-2 text-sm">
						<input type="checkbox" bind:checked={filterLottery} class="accent-blue-500" />
						隐藏抽奖弹幕
					</label>
				</div>

				<div>
					<label class="primary mb-1.5 block text-sm" for="obsUrl">浏览器源地址</label>
					<input
						id="obsUrl"
						readonly
						disabled
						value={obsUrl || '— 请先输入直播间号 —'}
						class="primary w-full cursor-default rounded border border-[#3a3a3a] bg-[#222] px-3 py-2 text-sm focus:outline-none disabled:opacity-100"
					/>
					<div class="mt-2 flex items-center justify-between">
						<p class="text-xs text-[#666]">将此地址粘贴到 OBS → 来源 → 浏览器。</p>
						<button
							onclick={copyUrl}
							disabled={!obsUrl}
							class="rounded bg-blue-600 px-4 py-1.5 text-sm font-medium text-white transition-colors hover:bg-blue-500 disabled:cursor-not-allowed disabled:opacity-40 {copied
								? 'bg-green-600 hover:bg-green-500'
								: ''}"
						>
							{copied ? '已复制！' : '复制'}
						</button>
					</div>
				</div>
			</div>
		</div>
	{:else if activeTab === 'backend'}
		<div class="mx-auto max-w-2xl px-6 py-8">
			<div class="mb-6 flex items-start justify-between gap-4">
				<div>
					<h2 class="secondary text-base font-semibold">后端配置</h2>
					<p class="mt-2 text-sm text-[#888]">更新配置后需要重启后端服务器。</p>
				</div>
				<button
					onclick={loadBackendConfig}
					disabled={backendLoading || backendSaving}
					class="rounded border border-[#3a3a3a] bg-[#262626] px-4 py-2 text-sm text-[#ddd] transition-colors hover:border-[#5a5a5a] hover:text-white disabled:cursor-not-allowed disabled:opacity-50"
				>
					{backendLoading ? '读取中...' : '重新读取'}
				</button>
			</div>

			{#if backendError}
				<div
					class="mb-4 rounded border border-red-500/40 bg-red-500/10 px-4 py-3 text-sm text-red-200"
				>
					{backendError}
				</div>
			{/if}

			{#if backendMessage}
				<div
					class="mb-4 rounded border border-green-500/40 bg-green-500/10 px-4 py-3 text-sm text-green-200"
				>
					{backendMessage}
				</div>
			{/if}

			<form
				class="space-y-5"
				onsubmit={(event) => {
					event.preventDefault();
					void saveBackendConfig();
				}}
			>
				<div class="grid gap-5 md:grid-cols-2">
					<div>
						<label class="primary mb-1.5 block text-sm" for="backendHost">监听地址</label>
						<input
							id="backendHost"
							type="text"
							placeholder="127.0.0.1 或 0.0.0.0"
							bind:value={backendHost}
							class="bg-color w-full rounded border border-[#3a3a3a] px-3 py-2 text-sm text-white placeholder-[#666] focus:border-blue-500 focus:outline-none"
						/>
						<p class="mt-1 text-xs text-[#666]">127.0.0.1 仅本机可访问，0.0.0.0 允许局域网访问。</p>
					</div>

					<div>
						<label class="primary mb-1.5 block text-sm" for="backendPort">端口</label>
						<input
							id="backendPort"
							type="number"
							min="1"
							max="65535"
							bind:value={backendPort}
							class="bg-color w-full rounded border border-[#3a3a3a] px-3 py-2 text-sm text-white placeholder-[#666] focus:border-blue-500 focus:outline-none"
						/>
					</div>
				</div>

				<div>
					<label class="primary mb-1.5 block text-sm" for="backendLogLevel">日志等级</label>
					<select
						id="backendLogLevel"
						bind:value={backendLogLevel}
						class="bg-color w-full rounded border border-[#3a3a3a] px-3 py-2 text-sm text-white focus:border-blue-500 focus:outline-none"
					>
						{#each LOG_LEVELS as level}
							<option value={level.value}>{level.label}</option>
						{/each}
					</select>
				</div>

				<div>
					<label class="primary mb-1.5 block text-sm" for="backendSessdata">SESSDATA</label>
					<div class="flex gap-2">
						<input
							id="backendSessdata"
							type={sessdataVisible ? 'text' : 'password'}
							placeholder="留空表示不写入配置文件"
							autocomplete="off"
							spellcheck="false"
							bind:value={backendSessdata}
							class="bg-color min-w-0 flex-1 rounded border border-[#3a3a3a] px-3 py-2 text-sm text-white placeholder-[#666] focus:border-blue-500 focus:outline-none"
						/>
						<button
							type="button"
							onclick={() => (sessdataVisible = !sessdataVisible)}
							class="rounded border border-[#3a3a3a] bg-[#262626] px-3 py-2 text-sm text-[#ddd] transition-colors hover:border-[#5a5a5a] hover:text-white"
						>
							{sessdataVisible ? '隐藏' : '显示'}
						</button>
					</div>
					<p class="mt-1 text-xs text-[#666]">
						SESSDATA 用于登录 B 站账号以获取更多弹幕。请勿泄露此值。
					</p>
					<p class="mt-1 text-xs text-[#666]">
						获取方法: 在浏览器开发者工具的「应用程序」→「存储」中找到 SESSDATA 的值。
					</p>
				</div>

				<div class="flex items-center justify-end gap-3">
					<button
						type="submit"
						disabled={backendLoading || backendSaving}
						class="rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-500 disabled:cursor-not-allowed disabled:opacity-50"
					>
						{backendSaving ? '保存中...' : '保存配置'}
					</button>
				</div>
			</form>
		</div>
	{:else if activeTab === 'tts'}
		<div class="mx-auto max-w-2xl px-6 py-8">
			<div class="mb-6 flex items-start justify-between gap-4">
				<div>
					<h2 class="secondary text-base font-semibold">语音播报（Windows TTS）</h2>
					<p class="mt-2 text-sm text-[#888]">配置更改会立即热更新，无需重启。仅 Windows 有效。</p>
				</div>
			</div>

			{#if ttsError}
				<div class="mb-4 rounded border border-red-500/40 bg-red-500/10 px-4 py-3 text-sm text-red-200">
					{ttsError}
				</div>
			{/if}
			{#if ttsMessage}
				<div class="mb-4 rounded border border-green-500/40 bg-green-500/10 px-4 py-3 text-sm text-green-200">
					{ttsMessage}
				</div>
			{/if}

			<form
				class="space-y-5"
				onsubmit={(event) => {
					event.preventDefault();
					void saveTTSConfig();
				}}
			>
				<!-- Enable toggle -->
				<label class="flex cursor-pointer items-center gap-3">
					<input type="checkbox" bind:checked={ttsEnabled} class="h-4 w-4 accent-blue-500" />
					<span class="primary text-sm font-medium">启用语音播报</span>
				</label>

				<div class="grid gap-5 md:grid-cols-2">
					<div>
						<label class="primary mb-1.5 block text-sm" for="ttsRoomId">直播间号</label>
						<input
							id="ttsRoomId"
							type="number"
							min="1"
							placeholder="例如: 213"
							bind:value={ttsRoomId}
							class="bg-color w-full rounded border border-[#3a3a3a] px-3 py-2 text-sm text-white placeholder-[#666] focus:border-blue-500 focus:outline-none"
						/>
					</div>
					<div>
						<label class="primary mb-1.5 block text-sm" for="ttsVoiceId">语音 ID（关键字）</label>
						<input
							id="ttsVoiceId"
							type="text"
							placeholder="ZH-CN"
							bind:value={ttsVoiceId}
							class="bg-color w-full rounded border border-[#3a3a3a] px-3 py-2 text-sm text-white placeholder-[#666] focus:border-blue-500 focus:outline-none"
						/>
						<p class="mt-1 text-xs text-[#666]">SAPI5 语音 ID 的匹配子串，启动时日志会列出可用语音。</p>
					</div>
				</div>

				<div class="grid gap-5 md:grid-cols-2">
					<div>
						<label class="primary mb-1.5 block text-sm" for="ttsRate">
							语速 <span class="text-[#888]">({ttsRate})</span>
						</label>
						<input
							id="ttsRate"
							type="range"
							min="-10"
							max="10"
							step="1"
							bind:value={ttsRate}
							class="w-full accent-blue-500"
						/>
						<div class="mt-1 flex justify-between text-xs text-[#666]">
							<span>-10（最慢）</span><span>0</span><span>10（最快）</span>
						</div>
					</div>
					<div>
						<label class="primary mb-1.5 block text-sm" for="ttsVolume">
							音量 <span class="text-[#888]">({ttsVolume})</span>
						</label>
						<input
							id="ttsVolume"
							type="range"
							min="0"
							max="100"
							step="1"
							bind:value={ttsVolume}
							class="w-full accent-blue-500"
						/>
						<div class="mt-1 flex justify-between text-xs text-[#666]">
							<span>0（静音）</span><span>100（最大）</span>
						</div>
					</div>
				</div>

				<div class="grid gap-5 md:grid-cols-2">
					<div>
						<label class="primary mb-1.5 block text-sm" for="ttsMaxQueue">最大队列长度</label>
						<input
							id="ttsMaxQueue"
							type="number"
							min="1"
							max="200"
							bind:value={ttsMaxQueue}
							class="bg-color w-full rounded border border-[#3a3a3a] px-3 py-2 text-sm text-white focus:border-blue-500 focus:outline-none"
						/>
						<p class="mt-1 text-xs text-[#666]">队列满时丢弃最旧的消息。</p>
					</div>
					<div>
						<label class="primary mb-1.5 block text-sm" for="ttsMaxAge">最大消息延迟（秒）</label>
						<input
							id="ttsMaxAge"
							type="number"
							min="0"
							bind:value={ttsMaxAgeSeconds}
							class="bg-color w-full rounded border border-[#3a3a3a] px-3 py-2 text-sm text-white focus:border-blue-500 focus:outline-none"
						/>
						<p class="mt-1 text-xs text-[#666]">播报前跳过超龄的消息。0 表示不跳过。</p>
					</div>
				</div>

				<hr class="border-[#3a3a3a]" />
				<h3 class="secondary text-sm font-medium">话语模板</h3>
				<p class="text-xs text-[#888]">占位符：{'{author_name}'} {'{content}'} {'{gift_name}'} {'{num}'} {'{price}'} {'{guard_name}'}</p>

				<div>
					<label class="primary mb-1.5 block text-sm" for="tmplText">普通弹幕</label>
					<input id="tmplText" type="text" bind:value={ttsTemplateText}
						class="bg-color w-full rounded border border-[#3a3a3a] px-3 py-2 text-sm text-white placeholder-[#666] focus:border-blue-500 focus:outline-none" />
				</div>
				<div>
					<label class="primary mb-1.5 block text-sm" for="tmplFreeGift">免费礼物</label>
					<input id="tmplFreeGift" type="text" bind:value={ttsTemplateFreeGift}
						class="bg-color w-full rounded border border-[#3a3a3a] px-3 py-2 text-sm text-white placeholder-[#666] focus:border-blue-500 focus:outline-none" />
				</div>
				<div>
					<label class="primary mb-1.5 block text-sm" for="tmplPaidGift">付费礼物</label>
					<input id="tmplPaidGift" type="text" bind:value={ttsTemplatePaidGift}
						class="bg-color w-full rounded border border-[#3a3a3a] px-3 py-2 text-sm text-white placeholder-[#666] focus:border-blue-500 focus:outline-none" />
				</div>
				<div>
					<label class="primary mb-1.5 block text-sm" for="tmplMember">舰长购买</label>
					<input id="tmplMember" type="text" bind:value={ttsTemplateMember}
						class="bg-color w-full rounded border border-[#3a3a3a] px-3 py-2 text-sm text-white placeholder-[#666] focus:border-blue-500 focus:outline-none" />
				</div>
				<div>
					<label class="primary mb-1.5 block text-sm" for="tmplSuperChat">醒目留言</label>
					<input id="tmplSuperChat" type="text" bind:value={ttsTemplateSuperChat}
						class="bg-color w-full rounded border border-[#3a3a3a] px-3 py-2 text-sm text-white placeholder-[#666] focus:border-blue-500 focus:outline-none" />
				</div>

				<div class="flex items-center justify-end gap-3">
					<button
						type="submit"
						disabled={ttsSaving}
						class="rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-500 disabled:cursor-not-allowed disabled:opacity-50"
					>
						{ttsSaving ? '保存中...' : '保存语音配置'}
					</button>
				</div>
			</form>
		</div>
	{:else if activeTab === 'preview'}
		<div class="flex h-[calc(100vh-45px)] flex-col items-center justify-center gap-4 p-6">
			{#if obsUrl}
				<div
					class="w-full max-w-2xl overflow-hidden rounded border border-[#3a3a3a] bg-black"
					style="aspect-ratio: 4/3;"
				>
					<iframe src={obsUrl} title="OBS 弹幕预览" class="h-full w-full border-0"></iframe>
				</div>
				<p class="text-xs text-[#666]">{obsUrl}</p>
			{:else}
				<p class="text-sm text-[#666]">请先在「配置」页填写直播间号。</p>
			{/if}
		</div>
	{:else if activeTab === 'help'}
		{#snippet icon(num: number)}
			<span
				class="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-blue-500 text-xs text-white"
				>{num}</span
			>
		{/snippet}
		<div class="mx-auto max-w-2xl px-6 py-8">
			<h2 class="secondary mb-5 text-base font-semibold">使用说明</h2>
			<ol class="primary space-y-4 text-sm">
				<li class="flex gap-3">
					{@render icon(1)}
					在<strong class="secondary">「配置」</strong>页输入 Bilibili 直播间号，选择喜欢的主题。
				</li>
				<li class="flex gap-3">
					{@render icon(2)}
					点击<strong class="secondary">「复制」</strong>按钮，复制生成的地址。
				</li>
				<li class="flex gap-3">
					{@render icon(3)}
					在 OBS 中添加<strong class="secondary">「浏览器」</strong
					>来源，粘贴地址，并设置合适的宽高。
				</li>
				<li class="flex gap-3">
					{@render icon(4)}
					弹幕将自动连接。可切换到<strong class="secondary">「预览」</strong>页确认效果。
				</li>
			</ol>
		</div>
	{:else if activeTab === 'about'}
		<div class="mx-auto max-w-2xl px-6 py-8">
			<h2 class="secondary mb-4 text-base font-semibold">关于</h2>
			<p class="primary text-sm">
				<strong class="secondary">bili-danmu-go</strong> — 基于 Go 的 Bilibili 直播弹幕转发服务器。
			</p>
			<p class="primary mt-3 text-sm">
				通过 Server-Sent Events 将弹幕实时推送到 OBS 浏览器源及插件。
			</p>
			<a
				href="https://github.com/iucario/bili-danmu-go"
				target="_blank"
				rel="noopener noreferrer"
				class="mt-5 inline-flex items-center gap-1.5 text-sm text-blue-400 hover:text-blue-300"
			>
				github.com/iucario/bili-danmu-go ↗
			</a>
		</div>
	{/if}
</div>
