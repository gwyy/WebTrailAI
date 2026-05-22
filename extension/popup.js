$(function() {
    if (typeof WebTrailAuth === 'undefined') {
        console.error('WebTrailAuth 未加载');
        return;
    }

    var authMode = 'login';
    var isShowingSummary = false;
    var currentLoggedIn = false;
    var VISITED_PAGES_KEY = 'visitedPages';
    var MAX_VISITED_PAGES_LENGTH = 50;
    var CLEAR_BUTTON_TEXT = '清空今日浏览历史（谨慎操作）';

    // =========================================================
    // UI 状态
    // =========================================================

    // 根据登录状态生成历史记录区域的空态提示。
    function getHistoryPlaceholderText(loggedIn) {
        return '暂无浏览记录';
    }

    // 渲染历史记录空态，避免列表区域出现空白。
    function renderHistoryPlaceholder(text) {
        $('#history-list').html(
            '<li class="history-placeholder"><p class="title">' + text + '</p></li>'
        );
    }

    function hasChromeStorage() {
        return typeof chrome !== 'undefined' && chrome.storage && chrome.storage.local;
    }

    // 读取本地最近浏览记录，弹窗列表不依赖任何后端查询。
    function getVisitedPages() {
        return new Promise(function(resolve) {
            if (hasChromeStorage()) {
                chrome.storage.local.get(VISITED_PAGES_KEY, function(data) {
                    var pages = data && data[VISITED_PAGES_KEY];
                    resolve(Array.isArray(pages) ? pages : []);
                });
                return;
            }

            if (typeof localStorage === 'undefined') {
                resolve([]);
                return;
            }

            try {
                var localPages = JSON.parse(localStorage.getItem(VISITED_PAGES_KEY) || '[]');
                resolve(Array.isArray(localPages) ? localPages : []);
            } catch (error) {
                resolve([]);
            }
        });
    }

    function setVisitedPages(pages) {
        return new Promise(function(resolve) {
            var safePages = Array.isArray(pages) ? pages.slice(0, MAX_VISITED_PAGES_LENGTH) : [];
            if (hasChromeStorage()) {
                var values = {};
                values[VISITED_PAGES_KEY] = safePages;
                chrome.storage.local.set(values, resolve);
                return;
            }

            if (typeof localStorage !== 'undefined') {
                localStorage.setItem(VISITED_PAGES_KEY, JSON.stringify(safePages));
            }
            resolve();
        });
    }

    function escapeHtml(value) {
        return String(value || '').replace(/[&<>"']/g, function(char) {
            return {
                '&': '&amp;',
                '<': '&lt;',
                '>': '&gt;',
                '"': '&quot;',
                "'": '&#39;'
            }[char];
        });
    }

    function formatVisitedTime(visitedAt) {
        var visitedDate = new Date(visitedAt);
        if (!visitedAt || isNaN(visitedDate.getTime())) {
            return '';
        }

        return visitedDate.toLocaleString('zh-CN', {
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit'
        });
    }

    function renderVisitedPages(loggedIn) {
        if (!$('#history-list').length) {
            return Promise.resolve();
        }

        return getVisitedPages().then(function(pages) {
            var visiblePages = pages.filter(function(page) {
                return page && page.url;
            }).slice(0, MAX_VISITED_PAGES_LENGTH);

            if (!visiblePages.length) {
                renderHistoryPlaceholder(getHistoryPlaceholderText(loggedIn));
                return;
            }

            var html = visiblePages.map(function(page) {
                var title = page.title || '无标题页面';
                var url = page.url || '';
                var visitedTime = formatVisitedTime(page.visitedAt);
                var timeHtml = visitedTime ? '<p class="visited-time">' + escapeHtml(visitedTime) + '</p>' : '';
                return [
                    '<li>',
                    '<p class="title" title="' + escapeHtml(title) + '">' + escapeHtml(title) + '</p>',
                    '<a href="' + escapeHtml(url) + '" title="' + escapeHtml(url) + '" target="_blank" rel="noreferrer">' + escapeHtml(url) + '</a>',
                    timeHtml,
                    '</li>'
                ].join('');
            }).join('');

            $('#history-list').html(html);
        });
    }

    function isSameLocalDay(leftDate, rightDate) {
        return leftDate.getFullYear() === rightDate.getFullYear() &&
            leftDate.getMonth() === rightDate.getMonth() &&
            leftDate.getDate() === rightDate.getDate();
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

    // 本地旧记录没有 visitedAt，无法判断日期，清空今日时一并移除旧格式记录。
    function removeTodayVisitedPages() {
        var today = new Date();
        return getVisitedPages().then(function(pages) {
            var remainingPages = pages.filter(function(page) {
                return !isTodayVisitedPage(page, today);
            });
            return setVisitedPages(remainingPages);
        });
    }

    function setClearButtonSubmitting(submitting) {
        var $btn = $('#clear-all-btn');
        $btn.prop('disabled', submitting);
        $btn.text(submitting ? '清空中...' : CLEAR_BUTTON_TEXT);
    }

    function showClearButtonError(message) {
        var $btn = $('#clear-all-btn');
        $btn.text(message || '清空失败，请稍后重试');
        setTimeout(function() {
            if (!$btn.prop('disabled')) {
                $btn.text(CLEAR_BUTTON_TEXT);
            }
        }, 2000);
    }

    // 按登录状态切换导航、总结入口、清空按钮和欢迎语。
    function updateLoginUI(loggedIn) {
        currentLoggedIn = loggedIn;
        var $loginNavItem = $('#login-nav-item');
        var $userNavItem = $('#user-nav-item');
        var $historyNavItem = $('#history-nav-item');
        var $clearAllBtn = $('#clear-all-btn');

        if (loggedIn) {
            $loginNavItem.hide();
            $userNavItem.show();
            $historyNavItem.show();
            $clearAllBtn.show();
            renderVisitedPages(true);

            WebTrailAuth.getStoredSession().then(function(session) {
                var name = session.username || '用户';
                $('#welcome-text').text(name + ' , 欢迎您');
            });
            return;
        }

        $loginNavItem.show();
        $userNavItem.hide();
        $historyNavItem.hide();
        $clearAllBtn.hide();
        $('.history-summary').hide();
        $('#content').show();
        $('#history-btn').text('每日总结');
        isShowingSummary = false;
        renderVisitedPages(false);
    }

    // 切换登录/注册模式，并同步标题、说明和提交按钮文案。
    function setAuthMode(mode) {
        authMode = mode;
        $('.auth-mode-btn').removeClass('active');
        $('.auth-mode-btn[data-mode="' + mode + '"]').addClass('active');
        hideAuthMessage();

        if (mode === 'register') {
            $('#auth-title').text('用户注册');
            $('#auth-subtitle').text('创建账号后将自动登录');
            $('#auth-submit-btn').text('立即注册');
            return;
        }

        $('#auth-title').text('用户登录');
        $('#auth-subtitle').text('请输入您的账号信息');
        $('#auth-submit-btn').text('立即登录');
    }

    // 打开鉴权弹窗并聚焦用户名输入框。
    function showAuthBox(mode) {
        setAuthMode(mode || 'login');
        $('#login-box').fadeIn(300);
        setTimeout(function() {
            $('#login-username').trigger('focus');
        }, 80);
    }

    // 关闭鉴权弹窗，保留当前输入内容方便用户修正后重试。
    function hideAuthBox() {
        $('#login-box').fadeOut(300);
    }

    // 清空登录/注册表单中的错误和成功提示。
    function hideAuthMessage() {
        $('#auth-error').hide().text('');
        $('#auth-success').hide().text('');
    }

    // 展示鉴权失败信息，并隐藏成功提示。
    function showAuthError(message) {
        $('#auth-success').hide().text('');
        $('#auth-error').text(message).show();
    }

    // 展示鉴权成功过程信息，并隐藏错误提示。
    function showAuthSuccess(message) {
        $('#auth-error').hide().text('');
        $('#auth-success').text(message).show();
    }

    // 将底层请求错误转换成用户可理解的提示文案。
    function getFriendlyError(error) {
        if (!error || !error.message || error.message === 'Failed to fetch') {
            return '无法连接后端服务，请确认 ' + WebTrailAuth.apiBaseUrl + ' 已启动';
        }
        return error.message;
    }

    // 根据提交状态禁用按钮，避免重复发起注册或登录请求。
    function setAuthSubmitting(submitting) {
        var $btn = $('#auth-submit-btn');
        $btn.prop('disabled', submitting);
        if (submitting) {
            $btn.text(authMode === 'register' ? '注册中...' : '登录中...');
            return;
        }
        $btn.text(authMode === 'register' ? '立即注册' : '立即登录');
    }

    // 清空用户名、密码和提示信息，通常在登录成功后执行。
    function clearAuthForm() {
        $('#login-username').val('');
        $('#login-password').val('');
        hideAuthMessage();
    }

    // 校验表单并发起登录或注册；注册成功后自动登录。
    function submitAuthForm() {
        var username = $('#login-username').val().trim();
        var password = $('#login-password').val();

        if (!username || !password) {
            showAuthError('用户名和密码不能为空');
            return;
        }

        if (!/^[a-z0-9_]{3,32}$/i.test(username)) {
            showAuthError('用户名只能包含字母、数字和下划线，长度为3到32位');
            return;
        }

        if (password.length < 6) {
            showAuthError('密码长度不能少于6位');
            return;
        }

        hideAuthMessage();
        setAuthSubmitting(true);

        var requestPromise;
        if (authMode === 'register') {
            requestPromise = WebTrailAuth.register(username, password).then(function(user) {
                showAuthSuccess('注册成功，正在登录...');
                return WebTrailAuth.login(user.username || username, password);
            });
        } else {
            requestPromise = WebTrailAuth.login(username, password);
        }

        requestPromise.then(function() {
            updateLoginUI(true);
            clearAuthForm();
            hideAuthBox();
        }).catch(function(error) {
            showAuthError(getFriendlyError(error));
        }).then(function() {
            setAuthSubmitting(false);
        });
    }

    // 页面载入时恢复登录状态
    WebTrailAuth.getValidAccessToken().then(function(token) {
        updateLoginUI(!!token);
    }).catch(function() {
        WebTrailAuth.clearSession().then(function() {
            updateLoginUI(false);
        });
    });

    if (typeof chrome !== 'undefined' && chrome.storage && chrome.storage.onChanged) {
        chrome.storage.onChanged.addListener(function(changes, areaName) {
            if (areaName === 'local' && changes[VISITED_PAGES_KEY]) {
                renderVisitedPages(currentLoggedIn);
            }
        });
    }

    // =========================================================
    // 初始化：隐藏除第一个之外的所有 summary
    // =========================================================
    $('.history-summary ul li').each(function(index) {
        if (index > 0) {
            $(this).find('.summary').hide();
            $(this).find('.summary-hide').text('展开');
            $(this).addClass('collapsed');
        }
    });

    // 1. 登录框的显示与隐藏
    $('#login-btn').on('click', function() {
        showAuthBox('login');
    });

    $('#login-close-btn').on('click', function() {
        hideAuthBox();
    });

    $('.auth-mode-btn').on('click', function() {
        setAuthMode($(this).data('mode'));
    });

    // 点击登录框外部关闭
    $('#login-box').on('click', function(e) {
        if (e.target.id === 'login-box') {
            hideAuthBox();
        }
    });

    $('#login-username, #login-password').on('keydown', function(e) {
        if (e.key === 'Enter') {
            submitAuthForm();
        }
    });

    // 2. 登录/注册提交
    $('#auth-submit-btn').on('click', function() {
        submitAuthForm();
    });

    // 3. 退出登录
    $('#logout-btn').on('click', function() {
        var $btn = $(this);
        $btn.prop('disabled', true).text('退出中...');
        WebTrailAuth.logout().then(function() {
            updateLoginUI(false);
        }).catch(function() {
            updateLoginUI(false);
        }).then(function() {
            $btn.prop('disabled', false).text('退出');
        });
    });

    // 4. 历史总结与历史记录的切换
    $('#history-btn').on('click', function() {
        if (!isShowingSummary) {
            $('.history-summary').show();
            $('#content').hide();
            $(this).text('返回历史记录');
            isShowingSummary = true;
            return;
        }

        $('.history-summary').hide();
        $('#content').fadeIn();
        $(this).text('每日总结');
        isShowingSummary = false;
    });

    // 5. 清空历史记录
    $('#clear-all-btn').on('click', function() {
        if (!confirm('确定要清空今日所有浏览历史吗？此操作不可恢复！')) {
            return;
        }

        var clearFailed = false;
        setClearButtonSubmitting(true);

        WebTrailAuth.requestProtectedJson('/api/cleanTodayTrail', {
            method: 'POST'
        }).then(function(res) {
            if (!res) {
                throw new Error('请先登录后清空');
            }
            return removeTodayVisitedPages();
        }).then(function() {
            $('#history-list').fadeOut(300, function() {
                renderVisitedPages(currentLoggedIn).then(function() {
                    $('#history-list').fadeIn(300);
                });
            });
        }).catch(function(error) {
            clearFailed = true;
            console.error('清空今日浏览历史失败:', error);
            showClearButtonError(error && error.message === '请先登录后清空' ? '请先登录后清空' : '清空失败，请稍后重试');
        }).then(function() {
            $('#clear-all-btn').prop('disabled', false);
            if (!clearFailed) {
                $('#clear-all-btn').text(CLEAR_BUTTON_TEXT);
            }
        });
    });

    // 6. 历史总结的展开收起功能
    $(document).on('click', '.summary-hide', function(e) {
        e.preventDefault();
        var $this = $(this);
        var $li = $this.closest('li');
        var $summary = $li.find('.summary');

        if ($summary.is(':visible')) {
            $summary.slideUp(300);
            $this.text('展开');
            $li.addClass('collapsed');
            return;
        }

        $summary.slideDown(300);
        $this.text('隐藏');
        $li.removeClass('collapsed');
    });

    // 7. 点击日期或查看更多打开详情页
    $(document).on('click', '.collapsed .date p, .summary-more', function(e) {
        e.preventDefault();
        e.stopPropagation();
        var dateText = $(this).data('time');

        if (typeof chrome !== 'undefined' && chrome.tabs) {
            chrome.tabs.create({
                url: chrome.runtime.getURL('detail.html') + '?date=' + encodeURIComponent(dateText)
            });
            return;
        }

        window.open('detail.html?date=' + encodeURIComponent(dateText), '_blank');
    });

    // 判断当前页面是否是 detail.html
    var currentPath = window.location.pathname;
    if (currentPath.endsWith('detail.html') || currentPath === '/detail.html' || currentPath === 'detail.html') {
        var queryString = window.location.search.substring(1);
        var params = {};
        var pairs = queryString.split('&');

        for (var i = 0; i < pairs.length; i++) {
            var pair = pairs[i].split('=');
            params[decodeURIComponent(pair[0])] = decodeURIComponent(pair[1] || '');
        }

        var dateValue = params.date;
        if (dateValue) {
            $('.detail-title').text(dateValue + ' 总结');
        }
    }
});
