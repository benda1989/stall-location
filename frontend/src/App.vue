<script>
export default {
  onLaunch() {
    // Customer mini-program entry is intentionally separate from merchant/admin H5.
  }
}
</script>

<style>
page {
  min-height: 100%;
  background: #fff7e8;
  color: #2f1f0d;
  font-family: "PingFang SC", "Microsoft YaHei", sans-serif;
  scrollbar-width: none;
}
button::after { border: 0; }

/* H5 preview scrolls on html/body/uni containers, not the scoped page node. */
html,
body,
#app,
uni-app,
uni-page,
uni-page-wrapper,
uni-page-body,
.uni-page-body {
  scrollbar-width: none;
  -ms-overflow-style: none;
}

html::-webkit-scrollbar,
body::-webkit-scrollbar,
#app::-webkit-scrollbar,
page::-webkit-scrollbar,
uni-app::-webkit-scrollbar,
uni-page::-webkit-scrollbar,
uni-page-wrapper::-webkit-scrollbar,
uni-page-body::-webkit-scrollbar,
.uni-page-body::-webkit-scrollbar,
scroll-view::-webkit-scrollbar {
  width: 0;
  height: 0;
  display: none;
  background: transparent;
}
</style>
