// 内容脚本提供当前页面的标题、URL 和正文文本；正文只用于后端上报。
chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
    if (request.action === 'getPageDetails') {
        const title = document.title;
        const url = window.location.href;
        const innerText = document.body ? document.body.innerText : '';
        sendResponse({
            title: title,
            url: url,
            innerText: innerText
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
