<script lang="ts">
	import { base } from '$app/paths';

	type Tab = 'config' | 'preview' | 'help' | 'about';

	let activeTab = $state<Tab>('config');

	// Form state
	let roomId = $state('');
	let theme = $state('plain');
	let filterLottery = $state(false);
	let copied = $state(false);

	const THEMES = [
		{ value: 'plain', label: '简洁' },
		{ value: 'bubble', label: '气泡' },
		{ value: 'bubble-light', label: '气泡（浅色）' }
	];

	const TAB_LABELS: Record<Tab, string> = {
		config: '配置',
		preview: '预览',
		help: '使用说明',
		about: '关于'
	};

	// Derived OBS URL
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
</script>

<div class="min-h-screen bg-[#1e1e1e] text-[#e3e3e3]">
	<!-- Tab bar -->
	<div class="flex justify-center border-b border-[#3a3a3a]">
		{#each (['config', 'preview', 'help', 'about'] as Tab[]) as tab}
			<button
				class="px-6 py-3 text-sm font-medium transition-colors
					{activeTab === tab
					? 'border-b-2 border-blue-400 text-white'
					: 'text-[#888] hover:text-[#ccc]'}"
				onclick={() => (activeTab = tab)}
			>
				{TAB_LABELS[tab]}
			</button>
		{/each}
	</div>

	<!-- Config tab -->
	{#if activeTab === 'config'}
		<div class="mx-auto max-w-lg px-6 py-8">
			<h2 class="mb-6 text-base font-semibold text-[#ccc]">OBS 浏览器源设置</h2>

			<div class="space-y-5">
				<!-- Room ID -->
				<div>
					<label class="mb-1.5 block text-sm text-[#aaa]" for="roomId">直播间号</label>
					<input
						id="roomId"
						type="number"
						min="1"
						placeholder="例如：12345"
						bind:value={roomId}
						class="w-full rounded border border-[#3a3a3a] bg-[#2a2a2a] px-3 py-2 text-sm
							text-white placeholder-[#555] focus:border-blue-500 focus:outline-none"
					/>
				</div>

				<!-- Theme -->
				<div>
					<p class="mb-1.5 text-sm text-[#aaa]">主题</p>
					<div class="flex gap-2">
						{#each THEMES as t}
							<button
								class="rounded border px-3 py-1.5 text-sm transition-colors
									{theme === t.value
									? 'border-blue-500 bg-blue-500/20 text-blue-300'
									: 'border-[#3a3a3a] bg-[#2a2a2a] text-[#aaa] hover:border-[#555]'}"
								onclick={() => (theme = t.value)}
							>
								{t.label}
							</button>
						{/each}
					</div>
				</div>

				<!-- Filters -->
				<div>
					<p class="mb-1.5 text-sm text-[#aaa]">过滤选项</p>
					<label class="flex cursor-pointer items-center gap-2 text-sm text-[#ccc]">
						<input type="checkbox" bind:checked={filterLottery} class="accent-blue-500" />
						隐藏抽奖弹幕
					</label>
				</div>

				<!-- Generated URL -->
				<div>
					<label class="mb-1.5 block text-sm text-[#aaa]" for="obsUrl">浏览器源地址</label>
					<input
						id="obsUrl"
						readonly
						disabled
						value={obsUrl || '— 请先输入直播间号 —'}
						class="w-full cursor-default rounded border border-[#3a3a3a] bg-[#222] px-3 py-2
							text-sm text-[#aaa] focus:outline-none disabled:opacity-100"
					/>
					<div class="mt-2 flex items-center justify-between">
						<p class="text-xs text-[#555]">将此地址粘贴到 OBS → 来源 → 浏览器。</p>
						<button
							onclick={copyUrl}
							disabled={!obsUrl}
							class="rounded bg-blue-600 px-4 py-1.5 text-sm font-medium text-white
								transition-colors hover:bg-blue-500 disabled:cursor-not-allowed
								disabled:opacity-40 {copied ? 'bg-green-600 hover:bg-green-500' : ''}"
						>
							{copied ? '已复制！' : '复制'}
						</button>
					</div>
				</div>
			</div>
		</div>

	<!-- Preview tab -->
	{:else if activeTab === 'preview'}
		<div class="flex h-[calc(100vh-45px)] flex-col items-center justify-center gap-4 p-6">
			{#if obsUrl}
				<div class="w-full max-w-2xl overflow-hidden rounded border border-[#3a3a3a] bg-black"
					style="aspect-ratio: 16/9;">
					<iframe src={obsUrl} title="OBS 弹幕预览" class="h-full w-full border-0"></iframe>
				</div>
				<p class="text-xs text-[#555]">{obsUrl}</p>
			{:else}
				<p class="text-sm text-[#555]">请先在「配置」页填写直播间号。</p>
			{/if}
		</div>

	<!-- Help tab -->
	{:else if activeTab === 'help'}
		<div class="mx-auto max-w-lg px-6 py-8">
			<h2 class="mb-5 text-base font-semibold text-[#ccc]">使用说明</h2>
			<ol class="space-y-4 text-sm text-[#aaa]">
				<li class="flex gap-3">
					<span class="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full
						bg-blue-600 text-xs text-white">1</span>
					在<strong class="text-[#ccc]">「配置」</strong>页输入 Bilibili 直播间号，选择喜欢的主题。
				</li>
				<li class="flex gap-3">
					<span class="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full
						bg-blue-600 text-xs text-white">2</span>
					点击<strong class="text-[#ccc]">「复制」</strong>按钮，复制生成的地址。
				</li>
				<li class="flex gap-3">
					<span class="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full
						bg-blue-600 text-xs text-white">3</span>
					在 OBS 中添加<strong class="text-[#ccc]">「浏览器」</strong>来源，粘贴地址，并设置合适的宽高。
				</li>
				<li class="flex gap-3">
					<span class="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full
						bg-blue-600 text-xs text-white">4</span>
					弹幕将自动连接。可切换到<strong class="text-[#ccc]">「预览」</strong>页确认效果。
				</li>
			</ol>
		</div>

	<!-- About tab -->
	{:else if activeTab === 'about'}
		<div class="mx-auto max-w-lg px-6 py-8">
			<h2 class="mb-4 text-base font-semibold text-[#ccc]">关于</h2>
			<p class="text-sm text-[#aaa]">
				<strong class="text-[#ccc]">bili-danmu-go</strong> — 基于 Go 的 Bilibili 直播弹幕转发服务器。
			</p>
			<p class="mt-3 text-sm text-[#aaa]">
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
