importScripts('auth.js');

var MAX_VISITED_PAGES_LENGTH = 50;
var MIN_TRAIL_TITLE_CHARS = 8;
var MAX_TRAIL_TITLE_CHARS = 512;
var DUPLICATE_PAGE_EVENT_TTL_MS = 3000;
var LEGACY_VISITED_PAGES_KEY = 'visitedPages';
var recentPageEvents = {};

// 只采集后端可接受的 http/https 页面，浏览器内部页和本地文件不进入流程。
function isUnsupportedUrl(url) {
    return !url || !/^https?:\/\//i.test(url);
}

// 后端要求标题不超过 512 个字符，前端先做一次轻量规整。
function normalizeTrailTitle(title) {
    var normalizedTitle = (title || '').trim();
    if (!normalizedTitle) {
        return '无标题页面';
    }

    var titleChars = Array.from(normalizedTitle);
    if (titleChars.length > MAX_TRAIL_TITLE_CHARS) {
        return titleChars.slice(0, MAX_TRAIL_TITLE_CHARS).join('');
    }
    return normalizedTitle;
}

function getTrailTitleLength(title) {
    return Array.from((title || '').trim()).length;
}

function isTrailTitleTooShort(title) {
    return getTrailTitleLength(title) < MIN_TRAIL_TITLE_CHARS;
}

function normalizeTrailInnerText(innerText) {
    return typeof innerText === 'string' ? innerText : '';
}

function isSameLocalDay(leftDate, rightDate) {
    return leftDate.getFullYear() === rightDate.getFullYear() &&
        leftDate.getMonth() === rightDate.getMonth() &&
        leftDate.getDate() === rightDate.getDate();
}

// 旧版本本地记录可能没有 visitedAt，无法还原日期，按当天处理以避免重复展示。
function isSameTrailDay(visitedAt, targetDate) {
    if (!visitedAt) {
        return true;
    }

    var visitedDate = new Date(visitedAt);
    if (isNaN(visitedDate.getTime())) {
        return true;
    }
    return isSameLocalDay(visitedDate, targetDate);
}

function isTodayVisitedPage(page, today) {
    if (!page || !page.visitedAt) {
        return true;
    }

    var visitedDate = new Date(page.visitedAt);
    if (isNaN(visitedDate.getTime())) {
        return true;
    }
    return isSameLocalDay(visitedDate, today);
}

// 本地历史只保留当天数据，跨天后首次采集会自动移除昨天及更早的记录。
function filterTodayVisitedPages(visitedPages, today) {
    if (!Array.isArray(visitedPages)) {
        return [];
    }

    return visitedPages.filter(function(page) {
        return isTodayVisitedPage(page, today);
    });
}

function hasDuplicateTrailTitleToday(visitedPages, page) {
    var title = (page.title || '').trim();
    var visitedDate = new Date(page.visitedAt || Date.now());
    return visitedPages.some(function(item) {
        return item &&
            (item.title || '').trim() === title &&
            isSameTrailDay(item.visitedAt, visitedDate);
    });
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
        innerText: normalizeTrailInnerText(result && result.innerText),
        visitedAt: Date.now()
    };
}

// 正文只发后端，不进入插件本地存储。
function buildLocalVisitedPage(page) {
    return {
        title: page.title,
        url: page.url,
        visitedAt: page.visitedAt
    };
}

function getVisitedPagesStorageKey(session) {
    return WebTrailAuth.buildUserScopedStorageKey(LEGACY_VISITED_PAGES_KEY, session);
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

// 维护最近浏览记录列表，同一 URL 会被移动到列表顶部；未登录时不记录本地历史。
function storeVisitedPage(page) {
    WebTrailAuth.getStoredSession().then(function(session) {
        return WebTrailAuth.getValidAccessToken().then(function(token) {
            if (!token) {
                cleanupExpiredVisitedPagesFromLocal(LEGACY_VISITED_PAGES_KEY, new Date(page.visitedAt || Date.now()));
                return;
            }

            var storageKey = getVisitedPagesStorageKey(session);
            if (!storageKey) {
                console.error('浏览历史缺少账号隔离存储键');
                return;
            }

            saveVisitedPageWhenLoggedIn(page, storageKey);
        });
    }).catch(function(error) {
        cleanupExpiredVisitedPagesFromLocal(LEGACY_VISITED_PAGES_KEY, new Date(page.visitedAt || Date.now()));
        console.error('校验登录态失败:', error);
    });
}

function saveVisitedPageWhenLoggedIn(page, storageKey) {
    chrome.storage.local.get(storageKey, function(data) {
        if (chrome.runtime.lastError) {
            console.error('读取本地浏览记录失败:', chrome.runtime.lastError);
            return;
        }

        WebTrailAuth.getStoredSession().then(function(session) {
            if (getVisitedPagesStorageKey(session) !== storageKey) {
                return;
            }

            var today = new Date(page.visitedAt || Date.now());
            var storedVisitedPages = Array.isArray(data[storageKey]) ? data[storageKey] : [];
            var visitedPages = filterTodayVisitedPages(storedVisitedPages, today);
            var shouldPersistCleanup = visitedPages.length !== storedVisitedPages.length;
            page.title = normalizeTrailTitle(page.title);
            var localPage = buildLocalVisitedPage(page);
            if (isTrailTitleTooShort(localPage.title)) {
                persistVisitedPagesCleanup(storageKey, visitedPages, shouldPersistCleanup);
                console.info('标题少于 8 个字符，已跳过本地浏览记录:', page.title);
                return;
            }

            if (hasDuplicateTrailTitleToday(visitedPages, localPage)) {
                persistVisitedPagesCleanup(storageKey, visitedPages, shouldPersistCleanup);
                console.info('当天标题重复，已跳过本地浏览记录:', page.title);
                return;
            }

            var existingIndex = visitedPages.findIndex(function(item) {
                return item.url === localPage.url;
            });

            if (existingIndex !== -1) {
                visitedPages.splice(existingIndex, 1);
            } else if (visitedPages.length >= MAX_VISITED_PAGES_LENGTH) {
                visitedPages.pop();
            }

            visitedPages.unshift(localPage);
            var latestPages = visitedPages.slice(0, MAX_VISITED_PAGES_LENGTH);
            WebTrailAuth.getStoredSession().then(function(latestSession) {
                if (getVisitedPagesStorageKey(latestSession) !== storageKey) {
                    console.info('登录账号已切换，已跳过保存本地浏览记录:', page.url);
                    return Promise.resolve(null);
                }

                return WebTrailAuth.getValidAccessToken();
            }).then(function(latestToken) {
                if (!latestToken) {
                    return;
                }

                var values = {};
                values[storageKey] = latestPages;
                chrome.storage.local.set(values, function() {
                    if (chrome.runtime.lastError) {
                        console.error('保存本地浏览记录失败:', chrome.runtime.lastError);
                        return;
                    }

                    syncVisitedPageToServer(page, storageKey);
                });
            }).catch(function(error) {
                console.error('校验登录态失败:', error);
            });
        }).catch(function(error) {
            console.error('校验登录态失败:', error);
        });
    });
}

function cleanupExpiredVisitedPagesFromLocal(storageKey, referenceDate) {
    chrome.storage.local.get(storageKey, function(data) {
        if (chrome.runtime.lastError) {
            console.error('读取本地浏览记录失败:', chrome.runtime.lastError);
            return;
        }

        var storedVisitedPages = Array.isArray(data[storageKey]) ? data[storageKey] : [];
        var visitedPages = filterTodayVisitedPages(storedVisitedPages, referenceDate);
        persistVisitedPagesCleanup(storageKey, visitedPages, visitedPages.length !== storedVisitedPages.length);
    });
}

function persistVisitedPagesCleanup(storageKey, visitedPages, shouldPersistCleanup) {
    if (!shouldPersistCleanup) {
        return;
    }

    var values = {};
    values[storageKey] = visitedPages.slice(0, MAX_VISITED_PAGES_LENGTH);
    chrome.storage.local.set(values, function() {
        if (chrome.runtime.lastError) {
            console.error('清理过期本地浏览记录失败:', chrome.runtime.lastError);
        }
    });
}

function removeVisitedPageFromLocal(page, storageKey) {
    chrome.storage.local.get(storageKey, function(data) {
        if (chrome.runtime.lastError) {
            console.error('读取本地浏览记录失败:', chrome.runtime.lastError);
            return;
        }

        var visitedPages = Array.isArray(data[storageKey]) ? data[storageKey] : [];
        var remainingPages = visitedPages.filter(function(item) {
            return !item || item.url !== page.url || item.visitedAt !== page.visitedAt;
        });

        var values = {};
        values[storageKey] = remainingPages;
        chrome.storage.local.set(values, function() {
            if (chrome.runtime.lastError) {
                console.error('移除本地浏览记录失败:', chrome.runtime.lastError);
            }
        });
    });
}

// 已登录时把浏览记录同步到 Go 后端。
function syncVisitedPageToServer(page, storageKey) {
    WebTrailAuth.getStoredSession().then(function(session) {
        if (getVisitedPagesStorageKey(session) !== storageKey) {
            console.info('登录账号已切换，已跳过同步浏览记录:', page.url);
            return null;
        }

        return WebTrailAuth.requestProtectedJson('/api/trailAdd', {
            method: 'POST',
            body: {
                title: page.title,
                url: page.url,
                innerText: page.innerText || ''
            }
        });
    }).then(function(res) {
        if (!res) {
            return;
        }
        if (res.data && res.data.filtered) {
            removeVisitedPageFromLocal(page, storageKey);
            console.info('浏览记录已被后端过滤:', res.data.reason);
            return;
        }
        console.info('浏览记录已同步到后端:', page.url);
    }).catch(function(error) {
        console.error('同步浏览记录失败:', error);
    });
}
