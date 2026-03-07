$(function(){

    // =========================================================
    // Auth 工具函数
    // =========================================================

    var AUTH_KEY = {
        jwt:            'wt_jwt_token',
        jwtExpires:     'wt_jwt_expires_at',
        refresh:        'wt_refresh_token',
        refreshExpires: 'wt_refresh_expires_at',
        username:       'wt_username',
    };

    var JWT_TTL_MS     = 60 * 60 * 1000;       // 1 小时
    var REFRESH_TTL_MS = 7 * 24 * 60 * 60 * 1000; // 7 天

    function saveTokens(data, username) {
        var now = Date.now();
        localStorage.setItem(AUTH_KEY.jwt,            data.jwt_token);
        localStorage.setItem(AUTH_KEY.jwtExpires,     now + JWT_TTL_MS);
        localStorage.setItem(AUTH_KEY.refresh,        data.refresh_token);
        localStorage.setItem(AUTH_KEY.refreshExpires, now + REFRESH_TTL_MS);
        if (username) localStorage.setItem(AUTH_KEY.username, username);
    }

    function clearTokens() {
        Object.values(AUTH_KEY).forEach(function(k) { localStorage.removeItem(k); });
    }

    function isExpired(expiresAt) {
        return !expiresAt || Date.now() >= parseInt(expiresAt, 10);
    }

    /**
     * 获取有效的 jwt_token。
     * - 若 jwt 未过期，直接返回。
     * - 若 jwt 过期但 refresh_token 有效，自动刷新并返回新 jwt。
     * - 若 refresh_token 也过期，返回 null（需重新登录）。
     *
     * 返回 Promise<string|null>
     */
    function getValidToken() {
        var jwt        = localStorage.getItem(AUTH_KEY.jwt);
        var jwtExp     = localStorage.getItem(AUTH_KEY.jwtExpires);
        var refreshTok = localStorage.getItem(AUTH_KEY.refresh);
        var refreshExp = localStorage.getItem(AUTH_KEY.refreshExpires);

        if (!jwt) return Promise.resolve(null);

        // jwt 仍有效
        if (!isExpired(jwtExp)) {
            return Promise.resolve(jwt);
        }

        // jwt 过期，尝试用 refresh_token 刷新
        if (!refreshTok || isExpired(refreshExp)) {
            clearTokens();
            updateLoginUI(false);
            return Promise.resolve(null);
        }

        return $.ajax({
            url:         'https://api.playwave.cc/user/token/refresh',
            type:        'POST',
            contentType: 'application/json',
            data:        JSON.stringify({ refresh_token: refreshTok }),
        }).then(function(res) {
            if (res && res.code === 1) {
                saveTokens(res);
                return res.jwt_token;
            }
            clearTokens();
            updateLoginUI(false);
            return null;
        }).catch(function() {
            clearTokens();
            updateLoginUI(false);
            return null;
        });
    }

    // =========================================================
    // UI 状态
    // =========================================================

    function updateLoginUI(loggedIn) {
        var $nav = $('ul.nav li');
        if (loggedIn) {
            var name = localStorage.getItem(AUTH_KEY.username) || '用户';
            $nav.eq(0).hide();
            $nav.eq(1).text(name + ' , 欢迎您').show();
        } else {
            $nav.eq(0).show();
            $nav.eq(1).hide();
        }
    }

    // 页面载入时恢复登录状态
    getValidToken().then(function(token) {
        updateLoginUI(!!token);
    });

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
        $('#login-error').hide();
        $('#login-box').fadeIn(300);
    });

    $('#login-close-btn').on('click', function() {
        $('#login-box').fadeOut(300);
    });

    // 点击登录框外部关闭
    $('#login-box').on('click', function(e) {
        if (e.target.id === 'login-box') {
            $(this).fadeOut(300);
        }
    });

    // 2. 登录提交
    $('#login-submit-btn').on('click', function() {
        var username = $('#login-username').val().trim();
        var password = $('#login-password').val().trim();

        if (!username || !password) {
            $('#login-error').text('用户名和密码不能为空').show();
            return;
        }

        var $btn = $(this);
        $btn.prop('disabled', true).text('登录中...');
        $('#login-error').hide();

        $.ajax({
            url:         'https://api.playwave.cc/user/login',
            type:        'POST',
            contentType: 'application/json',
            data:        JSON.stringify({ username: username, password: password }),
        }).done(function(res) {
            if (res && res.code === 1) {
                saveTokens(res, username);
                updateLoginUI(true);
                $('#login-box').fadeOut(300);
                $('#login-username').val('');
                $('#login-password').val('');
            } else {
                $('#login-error').text((res && res.msg) || '登录失败，请检查账号密码').show();
            }
        }).fail(function() {
            $('#login-error').text('网络错误，请稍后重试').show();
        }).always(function() {
            $btn.prop('disabled', false).text('立即登录');
        });
    });
    
    // 3. 历史总结与历史记录的切换
    let isShowingSummary = false;
    $('#history-btn').on('click', function() {
        if (!isShowingSummary) {
            // 显示历史总结
            $('.history-summary').show();
            $('#content').hide();
            $('#search').hide();
            $(this).text('返回历史记录');
            isShowingSummary = true;
        } else {
            // 返回历史记录
            $('.history-summary').hide();
            $('#content').fadeIn();
            $('#search').fadeIn();
            $(this).text('每日总结');
            isShowingSummary = false;
        }
    });
    
    // 4. 搜索功能
    function highlightText(text, keyword) {
        if (!keyword) return text;
        const regex = new RegExp(`(${keyword})`, 'gi');
        return text.replace(regex, '<mark>$1</mark>');
    }
    
    function performSearch() {
        const keyword = $('#search-input').val().trim();
        const $items = $('#history-list li');
        
        if (!keyword) {
            // 如果搜索框为空，显示所有项并移除高亮
            $items.show().each(function() {
                const $title = $(this).find('.title');
                const originalText = $title.data('original') || $title.text();
                $title.data('original', originalText);
                $title.html(originalText);
            });
            return;
        }
        
        $items.each(function() {
            const $item = $(this);
            const $title = $item.find('.title');
            const originalText = $title.data('original') || $title.text();
            $title.data('original', originalText);
            
            if (originalText.toLowerCase().includes(keyword.toLowerCase())) {
                $title.html(highlightText(originalText, keyword));
                $item.show();
            } else {
                $item.hide();
            }
        });
    }
    
    $('#search-btn').on('click', performSearch);
    
    $('#search-input').on('input', function() {
        performSearch();
    });
    
    $('#search-input').on('keypress', function(e) {
        if (e.which === 13) { // Enter键
            performSearch();
        }
    });
    
    // 5. 清空历史记录
    $('#clear-all-btn').on('click', function() {
        if (confirm('确定要清空今日所有浏览历史吗？此操作不可恢复！')) {
            $('#history-list').fadeOut(300, function() {
                $(this).empty().fadeIn(300);
            });
        }
    });
    
    // 6. 历史总结的展开收起功能
    $(document).on('click', '.summary-hide', function(e) {
        e.preventDefault();
        const $this = $(this);
        const $li = $this.closest('li');
        const $summary = $li.find('.summary');
        
        if ($summary.is(':visible')) {
            $summary.slideUp(300);
            $this.text('展开');
            $li.addClass('collapsed');
        } else {
            $summary.slideDown(300);
            $this.text('隐藏');
            $li.removeClass('collapsed');
        }
    });

    // 7. 点击日期或查看更多打开详情页
    $(document).on('click', '.collapsed .date p, .summary-more', function(e) {
        e.preventDefault();
        e.stopPropagation();
        dateText = $(this).data('time');
        // 使用 chrome.tabs.create 打开新标签页（浏览器扩展方式）
        if (typeof chrome !== 'undefined' && chrome.tabs) {
            chrome.tabs.create({
                url: chrome.runtime.getURL('detail.html') + '?date=' + encodeURIComponent(dateText)
            });
        } else {
            // 降级方案：使用 window.open
            window.open('detail.html?date=' + encodeURIComponent(dateText), '_blank');
        }
    });





    // 判断当前页面是否是 detail.html
    var currentPath = window.location.pathname;
    if (currentPath.endsWith('detail.html') || currentPath === '/detail.html' || currentPath === 'detail.html') {
        // 获取查询字符串
        var queryString = window.location.search.substring(1); // 去掉开头的 '?'

        // 将查询字符串解析为对象
        var params = {};
        var pairs = queryString.split('&');
        for (var i = 0; i < pairs.length; i++) {
            var pair = pairs[i].split('=');
            params[decodeURIComponent(pair[0])] = decodeURIComponent(pair[1] || '');
        }

        // 获取 date 的值
        var dateValue = params['date'];
        //请求后端 拿到这一天的总结数据
        
        
    } 
});