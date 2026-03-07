

//监听标签更新事件 当一个标签（tab）更新时触发。
//status:complete表示页面加载完成，tab.active 表示标签处于活动状态（即用户正在查看该标签）。
chrome.tabs.onUpdated.addListener((tabId, changeInfo, tab) => {
    if (tab.url.indexOf('chrome') === 0) {
        return;
    }
    if (changeInfo.status === 'complete' && tab.active) {
        console.log("received tab update event: " + tab.url);
        sendMessageToContentScript(tab);
    }
});
//当收到来自其他部分（比如内容脚本）的消息时触发。
//如果消息的 action 是 'contentScriptLoaded'（表示内容脚本已加载），则打印日志，并调用 sendMessageToContentScript 函数。
chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
    if (tab.url.indexOf('chrome') === 0) {
        return;
    }
    if (request.action === 'contentScriptLoaded') {
        console.log("received tab update event: " + sender.tab.url);
        sendMessageToContentScript(sender.tab);
    }
});
//向指定标签的内容脚本发送消息，action 为 'getPageDetails'。
//接收响应后，提取页面的标题（title）、内容（content）和 URL。
//如果有错误，打印错误日志。
//然后调用 storeVisitedPage 函数存储这些数据。
function sendMessageToContentScript(tab) {
    chrome.tabs.sendMessage(tab.id, { action: 'getPageDetails' }, (result) => {
        if (chrome.runtime.lastError) {
            console.error(chrome.runtime.lastError);
            return;
        }

        const pageData = {
            title: result.title,
            content: result.content,
            url: result.url
        };
        //将页面数据存储到本地存储（chrome.storage.local）中，并发送到服务器。
        storeVisitedPage(pageData)
    });
}
//从本地存储（chrome.storage.local）获取已访问页面的列表（visitedPages），如果不存在则初始化为空数组。
function storeVisitedPage(page) {
    //定义了一个最大长度（maxVisitedPagesLength）为 50，表示本地存储中最多只能保存 50 个访问页面的数据。
    chrome.storage.local.get('visitedPages', (data) => {
        const visitedPages = data.visitedPages || [];
        const maxVisitedPagesLength = 50;  //本地存储最大个数

        // Check if the page already exists in the list
        const existingIndex = visitedPages.findIndex((p) => p.url === page.url);

        // If the page exists, remove it from the list
        if (existingIndex !== -1) {
            visitedPages.splice(existingIndex, 1);
        } else if (visitedPages.length >= maxVisitedPagesLength) {
            // If the list is at the maximum length, remove the oldest page
            visitedPages.pop();
        }

        // Add the new page to the end of the list
        visitedPages.unshift(page);

        // Update the storage with the new list
        chrome.storage.local.set({ visitedPages: visitedPages });
    });
    //从本地存储中获取 token，如果存在且不为空，则调用 sendPageDataToServer 函数将页面数据发送到服务器。
    chrome.storage.local.get('token', (tokenData) => {
        let token = tokenData.token || "";
        //如果有 token 才发送信息到后端
        if (token && token != "") {
            sendPageDataToServer(token, page);
        }
    });

}

//使用 fetch API 发送 POST 请求到服务器 URL：'https://api.playwave.cc/tracer/upload'。
function sendPageDataToServer(token, page) {
    fetch('https://api.playwave.cc/tracer/upload', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': "Bearer " + token
        },
        body: JSON.stringify(page)
    })
        .then((response) => {
            if (response.ok) {
                console.log('Page data successfully sent to server');
            } else {
                console.error('Failed to send page data to server');
            }
        })
        .catch((error) => {
            console.error('Error:', error);
        });
}


/**
 tab:
active: true
audible: false
autoDiscardable: true
discarded: false
favIconUrl: "https://static.zhihu.com/heifetz/favicon.ico"
frozen: false
groupId: -1
height: 414
highlighted: true
id: 1469318342
incognito: false
index: 9
lastAccessed: 1772009813037.675
mutedInfo: {muted: false}
pinned: false
selected: true
splitViewId: -1
status: "complete"
title: "仅凭ai真的能做好复杂项目吗？ - 知乎"
url: "https://www.zhihu.com/question/1999041081275355787/answer/2009905093022082545"
width: 1920
windowId: 1469317312
 */