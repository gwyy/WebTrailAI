$(function(){
    // 初始化：隐藏除第一个之外的所有 summary
    $('.history-summary ul li').each(function(index) {
        if (index > 0) {
            $(this).find('.summary').hide();
            $(this).find('.summary-hide').text('展开');
            $(this).addClass('collapsed');
        }
    });
    
    // 1. 登录框的显示与隐藏
    $('#login-btn').on('click', function() {
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