<template>
  <main class="page">
    <section class="hero">
      <div class="eyebrow">Stall QR Store</div>
      <h1>出摊啦</h1>
      <p>摊主每天一键出摊，顾客扫码看位置、提前下单、到摊自提。</p>
    </section>
    <section class="grid">
      <RouterLink class="card link-card" to="/nearby">
        <span class="pill green">Map first</span>
        <h2>附近摊位地图</h2>
        <p class="muted">默认地图打开，标注多个营业中流动摊主，可切换列表。</p>
      </RouterLink>
      <RouterLink class="card link-card" to="/share/8K3XQ2">
        <span class="pill green">QR Mode</span>
        <h2>单摊二维码模式</h2>
        <p class="muted">只显示用户位置和目标摊主位置，不展示其他商贩。</p>
      </RouterLink>
      <RouterLink class="card link-card" to="/merchant/login">
        <span class="pill">Merchant</span>
        <h2>摊主登录</h2>
        <p class="muted">测试手机号 13800138000，验证码任意。</p>
      </RouterLink>
      <RouterLink class="card link-card" to="/admin">
        <span class="pill red">Admin</span>
        <h2>运营后台</h2>
        <p class="muted">默认账号 admin / admin123。</p>
      </RouterLink>
    </section>
  </main>
</template>

<style scoped>
.link-card { color: inherit; text-decoration: none; }
.link-card h2 { margin-bottom: 6px; }
</style>
