(function(global) {
    'use strict';

    var API_BASE_URL = 'https://webtrail.zmz8.com/';
    var REFRESH_TTL_MS = 7 * 24 * 60 * 60 * 1000;
    var EXPIRE_SKEW_MS = 30 * 1000;

    var AUTH_KEY = {
        accessToken: 'wt_access_token',
        accessExpires: 'wt_access_expires_at',
        refreshToken: 'wt_refresh_token',
        refreshExpires: 'wt_refresh_expires_at',
        tokenType: 'wt_token_type',
        username: 'wt_username',
        userId: 'wt_user_id'
    };

    var LEGACY_KEYS = [
        'token',
        'wt_jwt_token',
        'wt_jwt_expires_at',
        'wt_refresh_token',
        'wt_refresh_expires_at'
    ];

    // 判断当前运行环境是否支持 Chrome 扩展本地存储。
    function hasChromeStorage() {
        return typeof chrome !== 'undefined' && chrome.storage && chrome.storage.local;
    }

    // 汇总当前鉴权模块实际使用的存储键，方便统一读取和清理。
    function allAuthKeys() {
        return Object.keys(AUTH_KEY).map(function(name) {
            return AUTH_KEY[name];
        });
    }

    // 从扩展存储读取数据，非扩展环境降级到 localStorage，便于本地页面调试。
    function storageGet(keys) {
        return new Promise(function(resolve) {
            if (hasChromeStorage()) {
                chrome.storage.local.get(keys, function(data) {
                    resolve(data || {});
                });
                return;
            }

            var result = {};
            if (typeof localStorage !== 'undefined') {
                keys.forEach(function(key) {
                    result[key] = localStorage.getItem(key);
                });
            }
            resolve(result);
        });
    }

    // 写入扩展存储，非扩展环境降级到 localStorage。
    function storageSet(values) {
        return new Promise(function(resolve) {
            if (hasChromeStorage()) {
                chrome.storage.local.set(values, resolve);
                return;
            }

            if (typeof localStorage !== 'undefined') {
                Object.keys(values).forEach(function(key) {
                    localStorage.setItem(key, values[key]);
                });
            }
            resolve();
        });
    }

    // 删除指定存储键，退出登录或确认登录态失效时用于清理登录态。
    function storageRemove(keys) {
        return new Promise(function(resolve) {
            if (hasChromeStorage()) {
                chrome.storage.local.remove(keys, resolve);
                return;
            }

            if (typeof localStorage !== 'undefined') {
                keys.forEach(function(key) {
                    localStorage.removeItem(key);
                });
            }
            resolve();
        });
    }

    // 基于统一后端地址拼接接口路径，避免调用方散落硬编码 host。
    function buildUrl(path) {
        if (path.indexOf('http://') === 0 || path.indexOf('https://') === 0) {
            return path;
        }
        return API_BASE_URL.replace(/\/$/, '') + '/' + path.replace(/^\//, '');
    }

    // 安全解析 JSON 响应，兼容后端返回空内容或纯文本错误的情况。
    function parseJson(text) {
        if (!text) {
            return {};
        }
        try {
            return JSON.parse(text);
        } catch (error) {
            return { message: text };
        }
    }

    // 从不同响应结构中提取可展示错误信息，取不到时使用兜底文案。
    function getResponseMessage(data, fallback) {
        return (data && (data.message || data.msg || data.error)) || fallback;
    }

    // 区分后端不可用和登录态真实失效，避免本地调试重启服务时误退出登录。
    function isNetworkError(error) {
        return !!error && (
            error.isNetworkError === true ||
            error.name === 'TypeError' ||
            error.message === 'Failed to fetch' ||
            error.message === 'Load failed' ||
            error.message === 'NetworkError when attempting to fetch resource.'
        );
    }

    // 发送 JSON 请求并统一处理状态码、响应解析和 Bearer Token。
    function requestJson(path, options) {
        var requestOptions = options || {};
        var headers = requestOptions.headers || {};
        var fetchOptions = {
            method: requestOptions.method || 'GET',
            headers: headers
        };

        if (requestOptions.body !== undefined) {
            headers['Content-Type'] = 'application/json';
            fetchOptions.body = JSON.stringify(requestOptions.body);
        }

        if (requestOptions.token) {
            headers.Authorization = 'Bearer ' + requestOptions.token;
        }

        return fetch(buildUrl(path), fetchOptions).then(function(response) {
            return response.text().then(function(text) {
                var data = parseJson(text);
                if (!response.ok) {
                    throw new Error(getResponseMessage(data, '请求失败'));
                }
                return data;
            });
        }).catch(function(error) {
            if (isNetworkError(error)) {
                error.isNetworkError = true;
            }
            throw error;
        });
    }

    // 校验后端统一响应中的业务 code，HTTP 200 但业务失败时也要抛错。
    function ensureApiSuccess(res, fallback) {
        if (!res || res.code !== 0) {
            throw new Error(getResponseMessage(res, fallback));
        }
        return res;
    }

    // 发起需要登录态的 JSON 请求；没有可用 token 时返回 null，由调用方决定是否提示登录。
    function requestProtectedJson(path, options) {
        return getValidAccessToken().then(function(token) {
            if (!token) {
                return null;
            }

            var requestOptions = options || {};
            requestOptions.token = token;
            return requestJson(path, requestOptions).then(function(res) {
                return ensureApiSuccess(res, '请求失败');
            });
        });
    }

    // 规范化用户名展示和存储格式，与后端小写用户名规则保持一致。
    function normalizeUsername(username) {
        return (username || '').trim().toLowerCase();
    }

    // 根据登录用户生成账号隔离的本地存储键，避免不同账号共用浏览历史。
    function buildUserScopedStorageKey(baseKey, session) {
        var owner = session && (session.username || session.userId);
        var normalizedOwner = String(owner || '').trim().toLowerCase();
        if (!baseKey || !normalizedOwner) {
            return '';
        }
        return baseKey + ':' + encodeURIComponent(normalizedOwner);
    }

    // 判断时间戳是否过期，预留少量提前量避免临界点请求失败。
    function isExpired(expiresAt) {
        var expiresAtNumber = parseInt(expiresAt, 10);
        return !expiresAtNumber || Date.now() + EXPIRE_SKEW_MS >= expiresAtNumber;
    }

    // 保存登录响应中的 access token、refresh token 和用户信息。
    function saveSession(data, username, userId) {
        if (!data || !data.access_token) {
            throw new Error('登录响应缺少 access_token');
        }

        var now = Date.now();
        var expiresInSeconds = parseInt(data.expires_in, 10);
        if (!expiresInSeconds || expiresInSeconds <= 0) {
            expiresInSeconds = 3600;
        }

        var values = {};
        values[AUTH_KEY.accessToken] = data.access_token;
        values[AUTH_KEY.accessExpires] = String(now + expiresInSeconds * 1000);
        values[AUTH_KEY.tokenType] = data.token_type || 'Bearer';

        if (data.refresh_token) {
            values[AUTH_KEY.refreshToken] = data.refresh_token;
            values[AUTH_KEY.refreshExpires] = String(now + REFRESH_TTL_MS);
        }

        if (username) {
            values[AUTH_KEY.username] = normalizeUsername(username);
        }

        if (userId !== undefined && userId !== null) {
            values[AUTH_KEY.userId] = String(userId);
        }

        return storageSet(values).then(function() {
            return values;
        });
    }

    // 校验登录或刷新响应是否包含 access_token，不符合契约则抛出业务错误。
    function ensureTokenResponse(res, fallback) {
        if (!res || !res.access_token) {
            throw new Error(getResponseMessage(res, fallback));
        }
        return res;
    }

    // 清理当前登录态，同时移除历史版本遗留的 token 键。
    function clearSession() {
        var keys = allAuthKeys().concat(LEGACY_KEYS);
        return storageRemove(keys);
    }

    // 读取当前存储中的登录态，并补齐调用方需要的默认字段。
    function getStoredSession() {
        return storageGet(allAuthKeys()).then(function(data) {
            return {
                accessToken: data[AUTH_KEY.accessToken] || '',
                accessExpires: data[AUTH_KEY.accessExpires] || '',
                refreshToken: data[AUTH_KEY.refreshToken] || '',
                refreshExpires: data[AUTH_KEY.refreshExpires] || '',
                tokenType: data[AUTH_KEY.tokenType] || 'Bearer',
                username: data[AUTH_KEY.username] || '',
                userId: data[AUTH_KEY.userId] || ''
            };
        });
    }

    // 调用后端注册接口，成功时返回后端脱敏后的用户信息。
    function register(username, password) {
        return requestJson('/register', {
            method: 'POST',
            body: {
                username: username,
                password: password
            }
        }).then(function(res) {
            if (!res || res.code !== 0) {
                throw new Error(getResponseMessage(res, '注册失败'));
            }
            return res.data || {};
        });
    }

    // 调用后端登录接口，并把 gin-jwt 返回的 token 信息写入统一存储。
    function login(username, password) {
        return requestJson('/login', {
            method: 'POST',
            body: {
                username: username,
                password: password
            }
        }).then(function(res) {
            return saveSession(ensureTokenResponse(res, '登录失败，请检查账号密码'), username);
        });
    }

    // 使用 refresh token 刷新 access token；后端不可用时保留登录态，等服务恢复后再刷新。
    function refreshSession() {
        return getStoredSession().then(function(session) {
            if (!session.accessToken || !session.refreshToken || isExpired(session.refreshExpires)) {
                return clearSession().then(function() {
                    return null;
                });
            }

            return requestJson('/refresh_token', {
                method: 'POST',
                token: session.accessToken,
                body: {
                    refresh_token: session.refreshToken
                }
            }).then(function(res) {
                ensureTokenResponse(res, '登录状态已过期，请重新登录');
                return saveSession(res, session.username, session.userId).then(function() {
                    return res.access_token;
                });
            }).catch(function(error) {
                if (isNetworkError(error)) {
                    throw error;
                }
                return clearSession().then(function() {
                    return null;
                });
            });
        });
    }

    // 获取可用的 access token；若已过期则尝试自动刷新。
    function getValidAccessToken() {
        return getStoredSession().then(function(session) {
            if (!session.accessToken) {
                return null;
            }
            if (!isExpired(session.accessExpires)) {
                return session.accessToken;
            }
            return refreshSession();
        });
    }

    // 调用后端登出接口并清理本地登录态，网络失败也以本地退出为准。
    function logout() {
        return getStoredSession().then(function(session) {
            if (!session.accessToken) {
                return clearSession();
            }

            return requestJson('/logout', {
                method: 'POST',
                token: session.accessToken
            }).catch(function() {
                return null;
            }).then(function() {
                return clearSession();
            });
        });
    }

    global.WebTrailAuth = {
        keys: AUTH_KEY,
        apiBaseUrl: API_BASE_URL,
        buildUserScopedStorageKey: buildUserScopedStorageKey,
        buildUrl: buildUrl,
        clearSession: clearSession,
        getStoredSession: getStoredSession,
        getValidAccessToken: getValidAccessToken,
        isNetworkError: isNetworkError,
        isExpired: isExpired,
        login: login,
        logout: logout,
        requestProtectedJson: requestProtectedJson,
        register: register
    };
})(typeof self !== 'undefined' ? self : window);
