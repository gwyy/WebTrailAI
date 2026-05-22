importScripts('auth.js');

var MAX_VISITED_PAGES_LENGTH = 50;
var DUPLICATE_PAGE_EVENT_TTL_MS = 3000;
var recentPageEvents = {};

// 只采集后端可接受的 http/https 页面，浏览器内部页和本地文件不进入流程。
function isUnsupportedUrl(url) {
    return !url || !/^https?:\/\//i.test(url);
}

// 后端要求标题非空且不超过 512 个字符，前端先做一次轻量规整。
function normalizeTrailTitle(title) {
    var normalizedTitle = (title || '').trim();
    if (!normalizedTitle) {
        return '无标题页面';
    }

    var titleChars = Array.from(normalizedTitle);
    if (titleChars.length > 512) {
        return titleChars.slice(0, 512).join('');
    }
    return normalizedTitle;
}

// 同一页面加载时可能同时触发 tabs 和 content script 事件，短时间内只记录一次。
function shouldSkipDuplicatePageEvent(tabId, url) {
    var now = Date.now();
    var eventKey = String(tabId || '') + '|' + url;
    if (recentPageEvents[eventKey] && now - recentPageEvents[eventKey] < DUPLICATE_PAGE_EVENT_TTL_MS) {
        return true;
    }

    recentPageEvents[eventKey] = now;
    Object.keys(recentPageEvents).forEach(function(key) {
        if (now - recentPageEvents[key] > DUPLICATE_PAGE_EVENT_TTL_MS * 4) {
            delete recentPageEvents[key];
        }
    });
    return false;
}

function buildPageData(result, fallbackTab) {
    var pageUrl = (result && result.url) || (fallbackTab && fallbackTab.url);
    if (isUnsupportedUrl(pageUrl)) {
        return null;
    }

    return {
        title: normalizeTrailTitle((result && result.title) || (fallbackTab && fallbackTab.title)),
        url: pageUrl,
        visitedAt: Date.now()
    };
}

// 监听标签更新事件，页面加载完成且处于激活状态时采集页面信息。
chrome.tabs.onUpdated.addListener(function(tabId, changeInfo, tab) {
    if (isUnsupportedUrl(tab && tab.url)) {
        return;
    }
    if (changeInfo.status === 'complete' && tab.active) {
        console.log('received tab update event: ' + tab.url);
        collectPageDetails(tab);
    }
});

// 内容脚本加载完成后主动通知后台，后台再向当前页面请求详情。
chrome.runtime.onMessage.addListener(function(request, sender) {
    var senderTab = sender && sender.tab;
    if (!senderTab || isUnsupportedUrl(senderTab.url)) {
        return;
    }

    if (request.action === 'contentScriptLoaded') {
        console.log('received content script event: ' + senderTab.url);
        collectPageDetails(senderTab);
    }
});

// 监听单页应用的前端路由变化，补齐 history.pushState 场景下的新页面记录。
if (chrome.webNavigation && chrome.webNavigation.onHistoryStateUpdated) {
    chrome.webNavigation.onHistoryStateUpdated.addListener(function(details) {
        if (details.frameId !== 0 || isUnsupportedUrl(details.url)) {
            return;
        }

        chrome.tabs.get(details.tabId, function(tab) {
            if (chrome.runtime.lastError) {
                console.error(chrome.runtime.lastError);
                return;
            }
            collectPageDetails({
                id: details.tabId,
                title: tab && tab.title,
                url: details.url
            });
        });
    }, {
        url: [
            { schemes: ['http'] },
            { schemes: ['https'] }
        ]
    });
}

function collectPageDetails(tab) {
    if (!tab || isUnsupportedUrl(tab.url) || shouldSkipDuplicatePageEvent(tab.id, tab.url)) {
        return;
    }
    sendMessageToContentScript(tab);
}

// 向内容脚本请求当前页面详情，并在拿到结果后写入本地浏览记录。
function sendMessageToContentScript(tab) {
    chrome.tabs.sendMessage(tab.id, { action: 'getPageDetails' }, function(result) {
        if (chrome.runtime.lastError) {
            console.error(chrome.runtime.lastError);
            var fallbackPage = buildPageData(null, tab);
            if (fallbackPage) {
                storeVisitedPage(fallbackPage);
            }
            return;
        }

        var pageData = buildPageData(result, tab);
        if (!pageData) {
            return;
        }
        console.log(pageData);
        storeVisitedPage(pageData);
    });
}

// 维护最近浏览记录列表，同一 URL 会被移动到列表顶部。
function storeVisitedPage(page) {
    chrome.storage.local.get('visitedPages', function(data) {
        var visitedPages = Array.isArray(data.visitedPages) ? data.visitedPages : [];
        var existingIndex = visitedPages.findIndex(function(item) {
            return item.url === page.url;
        });

        if (existingIndex !== -1) {
            visitedPages.splice(existingIndex, 1);
        } else if (visitedPages.length >= MAX_VISITED_PAGES_LENGTH) {
            visitedPages.pop();
        }

        visitedPages.unshift(page);
        chrome.storage.local.set({ visitedPages: visitedPages.slice(0, MAX_VISITED_PAGES_LENGTH) });
    });

    syncVisitedPageToServer(page);
}

// 已登录时把浏览记录同步到 Go 后端；未登录时只保留本地记录。
function syncVisitedPageToServer(page) {
    WebTrailAuth.requestProtectedJson('/api/trailAdd', {
        method: 'POST',
        body: {
            title: page.title,
            url: page.url
        }
    }).then(function(res) {
        if (!res) {
            return;
        }
        console.info('浏览记录已同步到后端:', page.url);
    }).catch(function(error) {
        console.error('同步浏览记录失败:', error);
    });
}
