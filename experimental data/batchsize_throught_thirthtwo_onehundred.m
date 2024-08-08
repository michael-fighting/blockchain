%根据任务的变�? task 3500
x=[100,500,1000,5000,10000];%x轴上的数据，第一个�?�代表数据开始，第二个�?�代表间隔，第三个�?�代表终�?
a=[420,1368,2358,4457,6658]; %a数据y�?
b=[89,240,456,857,1789];
c=[78,190,346,721,1426]; 
% yyaxis left   
plot(x,a,'-*b',x,b,'-^r',x,c,'-og','markersize',8,'linewidth',2); %线�?�，颜色，标�?
axis([0,10200,0,8500])  %确定x轴与y轴框图大�?
set(gca,'XTick',(0:2000:10000)) %x轴范�?1-6，间�?1
set(gca,'YTick',(0:2000:8000)) %y轴范�?0-700，间�?100
h=legend('BumbleBee','WaterBear-QS-Q','WaterBear-QS-C','Location','Best');  %右上角标�?
% set(h,'Box','off');
set(gca,'fontsize',12);
xlabel('��������С','fontsize',12) %x轴坐标描�?
ylabel('������ (Tx/s)','fontsize',12) %y轴坐标描�?
% yyaxis right
% plot(x,c,'--p',x,d,'-*b','markersize',10,'linewidth',2); %线�?�，颜色，标�?
% axis([0,8,0,3.2])  %确定x轴与y轴框图大�?
% set(gca,'XTick',x) %x轴范�?1-6，间�?1
% set(gca,'XTickLabel',{'1/2','1','2','4','8'}); 
% set(gca,'YTick',(0:0.4:3.2)) %y轴范�?0-700，间�?100
% h=legend('Packaging time','Single delay','Location','Best');  %右上角标�?
% %set(h,'Box','off');
% set(gca,'fontsize',18);
% ylabel('Time(s)','fontsize',18) %y轴坐标描�?