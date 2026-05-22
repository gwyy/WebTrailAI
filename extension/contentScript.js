// 内容脚本只提供当前页面的标题和 URL，正文内容不进入本地存储或后端上报。
chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
    if (request.action === 'getPageDetails') {
        const title = document.title;
        const url = window.location.href;
        sendResponse({
            title: title,
            url: url
        });
    }
});

// 通知后台当前页面已经可以采集；document_idle 注入时 DOMContentLoaded 可能已经发生。
function notifyContentScriptLoaded() {
    chrome.runtime.sendMessage({ action: 'contentScriptLoaded' });
}

if (document.readyState === 'loading') {
    window.addEventListener('DOMContentLoaded', notifyContentScriptLoaded, { once: true });
} else {
    notifyContentScriptLoaded();
}
