/* eslint-disable @typescript-eslint/no-unused-vars */
/* eslint-disable @typescript-eslint/no-explicit-any */
import { useEffect, useState } from 'react';
import { teachersAPI } from '../../services/api';
import { useToast } from '../../components/Toast';

interface TeacherLessonHistory {
  teacher_id: string;
  teacher_name: string;
  lesson_id: string;
  lesson_title: string;
  lesson_slug: string;
  created_at: string;
  updated_at: string;
  status: string;
}

export default function TeacherHistory() {
  const [history, setHistory] = useState<TeacherLessonHistory[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const { error: showError, ToastContainer } = useToast();

  useEffect(() => {
    loadHistory();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const loadHistory = async () => {
    try {
      setLoading(true);
      const data = await teachersAPI.getHistory();
      setHistory(Array.isArray(data) ? data : []);
    } catch (err: any) {
      showError(err?.response?.data?.error || err?.message || 'Lỗi khi tải lịch sử');
      setHistory([]);
    } finally {
      setLoading(false);
    }
  };

  const filteredHistory = (history || []).filter(item =>
    item.teacher_name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    item.lesson_title.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const getStatusBadge = (status: string) => {
    const statusConfig: Record<string, { bg: string; text: string; icon: string }> = {
      'published': { bg: 'bg-emerald-100', text: 'text-emerald-800', icon: '📤' },
      'draft': { bg: 'bg-amber-100', text: 'text-amber-800', icon: '📝' },
      'archived': { bg: 'bg-gray-100', text: 'text-gray-800', icon: '📦' },
    };
    const config = statusConfig[status] || statusConfig['draft'];
    return (
      <span className={`inline-flex items-center gap-1 px-3 py-1 rounded-full text-xs font-semibold ${config.bg} ${config.text}`}>
        <span>{config.icon}</span>
        {status.charAt(0).toUpperCase() + status.slice(1)}
      </span>
    );
  };

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return new Intl.DateTimeFormat('vi-VN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    }).format(date);
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <div className="text-4xl mb-4">⏳</div>
          <p className="text-gray-600 font-semibold">Đang tải lịch sử...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-gradient-to-br from-gray-50 to-gray-100 p-6 md:p-10 min-h-screen">
      {/* Header */}
      <div className="mb-12">
        <h1 className="text-4xl md:text-5xl font-black text-gray-900 flex items-center gap-3">
          <span>📚</span>
          Lịch sử giảng viên
        </h1>
        <p className="text-lg text-gray-600 mt-2">
          Xem tất cả bài học được tạo/chỉnh sửa bởi giảng viên
        </p>
      </div>

      {/* Search */}
      <div className="mb-8">
        <input
          type="text"
          placeholder="Tìm kiếm giảng viên hoặc bài học..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          className="w-full bg-white border-2 border-indigo-300 p-4 rounded-xl focus:border-indigo-600 focus:outline-none text-gray-900 text-base font-semibold placeholder:text-gray-400 hover:border-indigo-400 transition-all duration-200 shadow-sm"
        />
      </div>

          {/* History Table */}
          
        

          {filteredHistory.length === 0 ? (
        <div className="bg-white rounded-2xl shadow-lg p-12 text-center">
          <div className="text-6xl mb-4">📭</div>
          <p className="text-gray-600 text-lg font-medium">Không có dữ liệu</p>
          <p className="text-gray-500 mt-2">Chưa có giảng viên nào tạo bài học</p>
        </div>
      ) : (
        <div className="bg-white rounded-2xl shadow-lg overflow-hidden border border-gray-100">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-gradient-to-r from-indigo-50 to-purple-50 border-b-2 border-indigo-200">
                <tr>
                  <th className="px-6 py-4 text-left text-sm font-bold text-gray-900">👨‍🏫 Giảng viên</th>
                  <th className="px-6 py-4 text-left text-sm font-bold text-gray-900">📝 Bài học</th>
                  <th className="px-6 py-4 text-left text-sm font-bold text-gray-900">📊 Trạng thái</th>
                  <th className="px-6 py-4 text-left text-sm font-bold text-gray-900">🕐 Tạo lúc</th>
                  <th className="px-6 py-4 text-left text-sm font-bold text-gray-900">✏️ Cập nhật lúc</th>
                </tr>
              </thead>
              <tbody>
                {filteredHistory.map((item, idx) => (
                  <tr
                    key={`${item.teacher_id}-${item.lesson_id}-${idx}`}
                    className="border-b border-gray-200 hover:bg-indigo-50 transition-colors duration-200"
                  >
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-3">
                        <div className="w-10 h-10 rounded-full bg-gradient-to-br from-blue-500 to-blue-600 flex items-center justify-center text-white font-bold text-sm">
                          {item.teacher_name.charAt(0).toUpperCase()}
                        </div>
                        <div>
                          <div className="text-sm font-bold text-gray-900">{item.teacher_name}</div>
                          <div className="text-xs text-gray-500">{item.teacher_id}</div>
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      <div>
                        <div className="text-sm font-semibold text-gray-900 max-w-xs truncate">{item.lesson_title}</div>
                        <div className="text-xs text-gray-500">{item.lesson_slug}</div>
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      {getStatusBadge(item.status)}
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-700 whitespace-nowrap">
                      {formatDate(item.created_at)}
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-700 whitespace-nowrap">
                      {formatDate(item.updated_at)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Footer Stats */}
          <div className="bg-gradient-to-r from-indigo-50 to-purple-50 px-6 py-4 border-t border-gray-200">
            <p className="text-sm font-semibold text-gray-700">
                              📊 Tổng: {filteredHistory.length} bài học {searchTerm && `(tìm kiếm: "${searchTerm}")`}
            </p>
          </div>
        </div>
      )}
      <ToastContainer />
    </div>
  );
}
