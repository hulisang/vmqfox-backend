import React from 'react'
import { QRCodeSVG } from 'qrcode.react'

interface QRCodeViewProps {
  url: string
  size?: number
  payType?: number // 1: 微信, 2: 支付宝
}

export const QRCodeView: React.FC<QRCodeViewProps> = ({ url, size = 200, payType }) => {
  return (
    <div className="relative p-3.5 bg-white rounded-2xl shadow-sm border border-border/80 flex items-center justify-center">
      <QRCodeSVG
        value={url}
        size={size}
        level="M"
        includeMargin={false}
        bgColor="#FFFFFF"
        fgColor="#000000"
      />
      {payType && (
        <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
          <div className="w-9 h-9 bg-white/90 backdrop-blur-xs rounded-lg p-1.5 shadow-sm border border-border/50 flex items-center justify-center">
            {payType === 1 ? (
              <span className="text-emerald-600 font-bold text-xs">微信</span>
            ) : (
              <span className="text-blue-600 font-bold text-xs">支</span>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
