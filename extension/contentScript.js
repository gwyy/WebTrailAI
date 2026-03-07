//这个文件是 Chrome 扩展的内容脚本（content script），它被注入到用户访问的网页中，可以直接访问网页的 DOM（文档对象模型）。它主要负责响应背景脚本的消息，并提供页面信息。下面逐部分解释代码：


//当收到来自背景脚本的消息时触发。 如果消息的 action 是 'getPageDetails'：
//获取当前页面的标题（document.title）。
//获取页面正文内容（document.body.innerText，只取文本，不包括 HTML 标签）。
//获取当前 URL（window.location.href）。
//通过 sendResponse 发送这些信息作为响应。

chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
    if (request.action === 'getPageDetails') {
        const title = document.title;
        const content = document.body.innerText;
        const url = window.location.href;

        sendResponse({
            title: title,
            content: content,
            url: url
        });
    }
});
//监听 DOM 加载事件（window.addEventListener('DOMContentLoaded')）：
//当网页的 DOM 加载完成时触发（不包括图像等资源）。
//向背景脚本发送消息，action 为 'contentScriptLoaded'。
//目的：通知背景脚本内容脚本已准备好，可以开始通信。
window.addEventListener('DOMContentLoaded', () => {
    chrome.runtime.sendMessage({ action: 'contentScriptLoaded' });
});